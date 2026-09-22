# JNI entry points and the generated Ebitengine view are looked up by name.
-keep class go.** { *; }
-keep class com.olivierh.bubblebobble.mobile.** { *; }
-keep class com.olivierh.bubblebobble.ebitenmobileview.** { *; }
-keepclasseswithmembernames,includedescriptorclasses class * { native <methods>; }
