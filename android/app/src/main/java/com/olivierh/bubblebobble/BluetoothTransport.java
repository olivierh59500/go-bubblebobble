package com.olivierh.bubblebobble;

import android.Manifest;
import android.annotation.SuppressLint;
import android.app.Activity;
import android.app.AlertDialog;
import android.bluetooth.BluetoothAdapter;
import android.bluetooth.BluetoothDevice;
import android.bluetooth.BluetoothManager;
import android.bluetooth.BluetoothServerSocket;
import android.bluetooth.BluetoothSocket;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.pm.PackageManager;
import android.os.Build;
import android.widget.ArrayAdapter;

import com.olivierh.bubblebobble.mobile.Mobile;

import java.io.Closeable;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;

/** Android owns discovery and RFCOMM. Go owns the authoritative game protocol.
 * A per-attempt token authenticates each private loopback connection; it never
 * travels over Bluetooth. Generation IDs reject callbacks from cancelled UI. */
@SuppressLint("MissingPermission")
final class BluetoothTransport implements AutoCloseable {
    private static final UUID SERVICE = UUID.fromString("42554242-4c45-424f-4242-4c4500000001");
    private final Activity activity;
    private final BluetoothAdapter adapter;
    private final ExecutorService workers = Executors.newCachedThreadPool();
    private final Object guard = new Object();
    private final List<Closeable> sockets = new ArrayList<>();
    private final Map<Integer, Request> requests = new HashMap<>();
    private long generation;
    private boolean stopped = true;
    private boolean destroyed;
    private boolean host;
    private int port;
    private byte[] secret;
    private int nextRequest = 5000;
    private BroadcastReceiver discovery;
    private AlertDialog picker;
    private ArrayAdapter<String> labels;
    private final List<BluetoothDevice> devices = new ArrayList<>();

    private static final class Request {
        final long generation;
        final String stage;
        Request(long generation, String stage) { this.generation = generation; this.stage = stage; }
    }

    BluetoothTransport(Activity activity) {
        this.activity = activity;
        BluetoothManager manager = (BluetoothManager) activity.getSystemService(Context.BLUETOOTH_SERVICE);
        adapter = manager == null ? null : manager.getAdapter();
    }
    boolean available() { return adapter != null; }

    void command(String command) {
        String[] parts = command.split("\\|", -1);
        try {
            long id = Long.parseLong(parts[1]);
            if (parts[0].equals("BT_CANCEL")) {
                if (id >= generation) stop();
                return;
            }
            if (id < generation || destroyed) return;
            stop();
            boolean hosting = parts[0].equals("BT_HOST");
            if ((!hosting && !parts[0].equals("BT_JOIN")) || parts.length != (hosting ? 4 : 3)) throw new IllegalArgumentException();
            byte[] token = decodeToken(parts[hosting ? 3 : 2]);
            int endpoint = hosting ? Integer.parseInt(parts[2]) : 0;
            if (hosting && (endpoint < 1 || endpoint > 65535)) throw new IllegalArgumentException();
            synchronized (guard) {
                generation = id; stopped = false; host = hosting; port = endpoint; secret = token;
            }
            if (!available()) { fail(id, "This phone has no Bluetooth adapter"); return; }
            String[] missing = permissions();
            if (missing.length == 0) enable(id);
            else activity.requestPermissions(missing, request(id, "permissions"));
        } catch (RuntimeException error) { fail(generation, "Could not prepare Bluetooth"); }
    }

    private static byte[] decodeToken(String text) {
        if (!text.matches("[0-9a-f]{32}")) throw new IllegalArgumentException();
        byte[] token = new byte[16];
        for (int i = 0; i < token.length; i++) token[i] = (byte) Integer.parseInt(text.substring(i * 2, i * 2 + 2), 16);
        return token;
    }

    private int request(long id, String stage) {
        while (requests.containsKey(nextRequest)) nextRequest = nextRequest == 65000 ? 5000 : nextRequest + 1;
        int code = nextRequest;
        nextRequest = nextRequest == 65000 ? 5000 : nextRequest + 1;
        requests.put(code, new Request(id, stage));
        return code;
    }

    private String[] permissions() {
        List<String> result = new ArrayList<>();
        if (Build.VERSION.SDK_INT >= 31) {
            missing(result, Manifest.permission.BLUETOOTH_CONNECT);
            missing(result, host ? Manifest.permission.BLUETOOTH_ADVERTISE : Manifest.permission.BLUETOOTH_SCAN);
        } else if (!host) missing(result, Manifest.permission.ACCESS_FINE_LOCATION);
        return result.toArray(new String[0]);
    }
    private void missing(List<String> result, String name) {
        if (activity.checkSelfPermission(name) != PackageManager.PERMISSION_GRANTED) result.add(name);
    }
    void permissionResult(int code) {
        Request r = requests.remove(code);
        if (r == null || !current(r.generation)) return;
        if (permissions().length != 0) fail(r.generation, "Allow nearby devices to play over Bluetooth");
        else enable(r.generation);
    }
    private void enable(long id) {
        if (!current(id)) return;
        try {
            if (adapter.isEnabled()) chooseMode(id);
            else activity.startActivityForResult(new Intent(BluetoothAdapter.ACTION_REQUEST_ENABLE), request(id, "enable"));
        } catch (RuntimeException error) { fail(id, "Bluetooth could not be enabled"); }
    }
    void activityResult(int code, int result) {
        Request r = requests.remove(code);
        if (r == null || !current(r.generation)) return;
        if (r.stage.equals("enable")) {
            if (result == Activity.RESULT_OK) chooseMode(r.generation);
            else fail(r.generation, "Bluetooth was not enabled");
        } else if (r.stage.equals("visible")) {
            if (result > 0) listen(r.generation);
            else fail(r.generation, "Discoverability was cancelled");
        }
    }
    private void chooseMode(long id) {
        if (!current(id)) return;
        try {
        if (host) {
            Intent intent = new Intent(BluetoothAdapter.ACTION_REQUEST_DISCOVERABLE);
            intent.putExtra(BluetoothAdapter.EXTRA_DISCOVERABLE_DURATION, 180);
            activity.startActivityForResult(intent, request(id, "visible"));
        } else chooseDevice(id);
        } catch (RuntimeException error) { fail(id, "Bluetooth is unavailable. Check permissions and try again."); }
    }

    private void listen(long id) {
        final int endpoint = port;
        final byte[] token = secret.clone();
        status(id, "Waiting for the second player");
        async(() -> {
            try {
                BluetoothServerSocket server = adapter.listenUsingRfcommWithServiceRecord("Bubble Bobble Co-op", SERVICE);
                if (!own(id, server)) return;
                BluetoothSocket remote = server.accept(180000);
                server.close();
                if (!own(id, remote)) return;
                Socket local = new Socket();
                if (!own(id, local)) return;
                local.connect(new InetSocketAddress(InetAddress.getByName("127.0.0.1"), endpoint), 5000);
                local.setTcpNoDelay(true);
                local.getOutputStream().write(token); local.getOutputStream().flush();
                pipe(id, remote, local);
            } catch (IOException | SecurityException error) { fail(id, "Bluetooth hosting stopped. Please try again."); }
            finally { Arrays.fill(token, (byte) 0); }
        });
    }

    private void chooseDevice(long id) {
        stopDiscovery(); devices.clear();
        labels = new ArrayAdapter<>(activity, android.R.layout.simple_list_item_1);
        for (BluetoothDevice device : adapter.getBondedDevices()) addDevice(device);
        picker = new AlertDialog.Builder(activity).setTitle("Join Bubble Bobble")
            .setAdapter(labels, (dialog, which) -> {
                BluetoothDevice selected = devices.get(which);
                stopDiscovery();
                if (current(id)) connect(id, selected);
            })
            .setNegativeButton("Cancel", (dialog, which) -> fail(id, "Bluetooth connection cancelled"))
            .setOnCancelListener(dialog -> fail(id, "Bluetooth connection cancelled"))
            .create();
        picker.show();
        discovery = new BroadcastReceiver() {
            @Override public void onReceive(Context context, Intent intent) {
                if (!current(id)) return;
                if (BluetoothDevice.ACTION_FOUND.equals(intent.getAction())) {
                    BluetoothDevice device = intent.getParcelableExtra(BluetoothDevice.EXTRA_DEVICE);
                    if (device != null) addDevice(device);
                } else if (BluetoothAdapter.ACTION_DISCOVERY_FINISHED.equals(intent.getAction())) {
                    status(id, devices.isEmpty() ? "No phones found. Host on the other phone, then try again." : "Select the phone hosting your game");
                }
            }
        };
        IntentFilter filter = new IntentFilter(BluetoothDevice.ACTION_FOUND);
        filter.addAction(BluetoothAdapter.ACTION_DISCOVERY_FINISHED);
        if (Build.VERSION.SDK_INT >= 33) activity.registerReceiver(discovery, filter, Context.RECEIVER_EXPORTED);
        else activity.registerReceiver(discovery, filter);
        adapter.cancelDiscovery();
        if (adapter.startDiscovery()) status(id, "Searching for nearby phones");
        else status(id, "Choose a paired phone, or enable Bluetooth discovery");
    }

    private void addDevice(BluetoothDevice device) {
        for (BluetoothDevice existing : devices) if (existing.getAddress().equals(device.getAddress())) return;
        if (devices.size() >= 64) return;
        devices.add(device);
        String name = device.getName();
        labels.add((name == null ? "Android device" : name) + "\n" + device.getAddress());
    }

    private void connect(long id, BluetoothDevice device) {
        final byte[] token = secret.clone();
        status(id, "Connecting to the other phone");
        async(() -> {
            try {
                BluetoothSocket remote = device.createRfcommSocketToServiceRecord(SERVICE);
                if (!own(id, remote)) return;
                remote.connect();
                ServerSocket server = new ServerSocket();
                if (!own(id, server)) return;
                server.bind(new InetSocketAddress(InetAddress.getByName("127.0.0.1"), 0), 1);
                server.setSoTimeout(15000);
                final String address = "127.0.0.1:" + server.getLocalPort();
                activity.runOnUiThread(() -> { if (current(id)) Mobile.bluetoothClientReady(id, address, "HOST"); });
                Socket local = server.accept(); server.close();
                if (!own(id, local)) return;
                local.setTcpNoDelay(true); local.setSoTimeout(5000);
                byte[] supplied = new byte[16];
                readToken(local.getInputStream(), supplied);
                boolean valid = MessageDigest.isEqual(token, supplied); Arrays.fill(supplied, (byte) 0);
                if (!valid) throw new IOException("Invalid proxy token");
                local.setSoTimeout(0);
                pipe(id, remote, local);
            } catch (IOException | SecurityException error) { fail(id, "Could not join. Check that the other phone is hosting."); }
            finally { Arrays.fill(token, (byte) 0); }
        });
    }

    private static void readToken(InputStream stream, byte[] token) throws IOException {
        int offset = 0;
        while (offset < token.length) {
            int n = stream.read(token, offset, token.length - offset);
            if (n < 0) throw new IOException("Proxy closed before authentication");
            offset += n;
        }
    }
    private boolean own(long id, Closeable socket) {
        synchronized (guard) { if (current(id)) { sockets.add(socket); return true; } }
        closeSocket(socket); return false;
    }
    private void pipe(long id, BluetoothSocket remote, Socket local) throws IOException {
        AtomicBoolean ended = new AtomicBoolean();
        Runnable finish = () -> { if (ended.compareAndSet(false, true)) {closeSocket(remote);closeSocket(local);fail(id, "Bluetooth connection closed");} };
        InputStream incoming = remote.getInputStream(); OutputStream outgoing = remote.getOutputStream();
        InputStream gameIn = local.getInputStream(); OutputStream gameOut = local.getOutputStream();
        async(() -> copy(incoming, gameOut, finish));
        async(() -> copy(gameIn, outgoing, finish));
        status(id, "Bluetooth connected");
    }
    private static void copy(InputStream in, OutputStream out, Runnable finish) {
        byte[] buffer = new byte[8192];
        try { int count; while ((count = in.read(buffer)) >= 0) { if (count > 0) {out.write(buffer, 0, count);out.flush();} } }
        catch (IOException ignored) { /* Closing a socket cancels both directions. */ }
        finally { finish.run(); }
    }

    private void async(Runnable task) {synchronized (guard) {if (!destroyed) workers.execute(task);}}

    private boolean current(long id) { synchronized (guard) { return !destroyed && !stopped && generation == id; } }
    private void status(long id, String message) { activity.runOnUiThread(() -> {if (current(id)) Mobile.bluetoothStatus(id, message);}); }
    private void fail(long id, String message) {
        activity.runOnUiThread(() -> { if (current(id)) {Mobile.bluetoothFailed(id, message);stop();} });
    }
    private void stopDiscovery() {
        if (discovery != null) { try {activity.unregisterReceiver(discovery);} catch (IllegalArgumentException ignored) { } discovery = null; }
        if (picker != null) { picker.dismiss();picker = null; }
        if (adapter != null && (Build.VERSION.SDK_INT < 31 || activity.checkSelfPermission(Manifest.permission.BLUETOOTH_SCAN) == PackageManager.PERMISSION_GRANTED)) {
            try { adapter.cancelDiscovery(); } catch (SecurityException ignored) { }
        }
    }
    private void stop() {
        List<Closeable> old;
        synchronized (guard) {
            stopped = true; old = new ArrayList<>(sockets); sockets.clear();
            if (secret != null) {Arrays.fill(secret, (byte) 0);secret = null;}
        }
        stopDiscovery();
        for (Closeable socket : old) closeSocket(socket);
    }
    private static void closeSocket(Closeable socket) {try {socket.close();} catch (IOException ignored) { }}
    @Override public void close() { synchronized (guard) {destroyed = true;}stop();workers.shutdownNow(); }
}
