package com.olivierh.bubblebobble;

import android.app.Activity;
import android.os.Build;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.view.View;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;

import com.olivierh.bubblebobble.mobile.EbitenView;
import com.olivierh.bubblebobble.mobile.Mobile;
import go.Seq;

public final class MainActivity extends Activity {
    private EbitenView view;
    private BluetoothTransport transport;
    private final Handler handler = new Handler(Looper.getMainLooper());
    private final Runnable poll = new Runnable() {
        @Override public void run() {
            for (int i = 0; i < 8; i++) {
                String command = Mobile.pollBluetoothCommand();
                if (command == null || command.isEmpty()) break;
                if (command.equals("APP_EXIT")) { finish(); return; }
                transport.command(command);
            }
            handler.postDelayed(this, 50);
        }
    };

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        Seq.setContext(getApplicationContext());
        Mobile.setFilesDir(getFilesDir().getAbsolutePath());
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        if (Build.VERSION.SDK_INT >= 28) {
            WindowManager.LayoutParams p = getWindow().getAttributes();
            p.layoutInDisplayCutoutMode = WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
            getWindow().setAttributes(p);
        }
        transport = new BluetoothTransport(this);
        Mobile.setBluetoothAvailable(transport.available());
        view = new EbitenView(this);
        view.setFocusableInTouchMode(true);
        view.requestFocus();
        setContentView(view);
        immersive();
        handler.post(poll);
        if (Build.VERSION.SDK_INT >= 33) {
            getOnBackInvokedDispatcher().registerOnBackInvokedCallback(
                android.window.OnBackInvokedDispatcher.PRIORITY_DEFAULT, Mobile::back);
        }
    }

    @Override protected void onPause() {
        Mobile.suspend();
        if (view != null) view.suspendGame();
        super.onPause();
    }
    @Override protected void onResume() {
        super.onResume();
        if (view != null) { immersive(); view.resumeGame(); }
    }
    @Override public void onWindowFocusChanged(boolean focused) {
        super.onWindowFocusChanged(focused);
        if (focused && view != null) immersive();
    }
    @Override public void onBackPressed() { Mobile.back(); }
    @Override public void onRequestPermissionsResult(int code, String[] permissions, int[] results) {
        super.onRequestPermissionsResult(code, permissions, results);
        transport.permissionResult(code);
    }
    @Override protected void onActivityResult(int code, int result, android.content.Intent data) {
        super.onActivityResult(code, result, data);
        transport.activityResult(code, result);
    }
    @Override protected void onDestroy() {
        handler.removeCallbacks(poll);
        Mobile.cancelBluetooth();
        if (transport != null) transport.close();
        super.onDestroy();
    }

    private void immersive() {
        if (Build.VERSION.SDK_INT >= 30) {
            getWindow().setDecorFitsSystemWindows(false);
            WindowInsetsController controller = getWindow().getDecorView().getWindowInsetsController();
            if (controller != null) {
                controller.hide(WindowInsets.Type.statusBars() | WindowInsets.Type.navigationBars());
                controller.setSystemBarsBehavior(WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
            }
        } else {
            getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_FULLSCREEN
                | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
                | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
                | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
        }
    }
}
