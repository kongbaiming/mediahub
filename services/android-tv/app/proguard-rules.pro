# MediaHub Android TV ProGuard rules

# ─── Kotlin / Compose ───
-dontwarn kotlin.**
-keep class kotlin.Metadata { *; }
-keep class kotlin.coroutines.Continuation
-keepclassmembers class kotlinx.coroutines.** { volatile <fields>; }

# ─── kotlinx.serialization ───
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.AnnotationsKt
-keep,includedescriptorclasses class com.mediahub.tv.**$$serializer { *; }
-keepclassmembers class com.mediahub.tv.** {
    *** Companion;
}
-keepclasseswithmembers class com.mediahub.tv.** {
    kotlinx.serialization.KSerializer serializer(...);
}

# ─── Coil ───
-dontwarn coil.**
-keep class coil.** { *; }

# ─── Media3 / ExoPlayer ───
-keep class androidx.media3.** { *; }
-dontwarn androidx.media3.**

# ─── OkHttp ───
-dontwarn okhttp3.**
-dontwarn okio.**

# ─── AndroidX Leanback ───
-keep class androidx.leanback.** { *; }
-dontwarn androidx.leanback.**

# ─── Compose for TV ───
-keep class androidx.tv.** { *; }
-dontwarn androidx.tv.**

# ─── 应用模型（kotlinx.serialization） ───
-keep class com.mediahub.tv.data.model.** { *; }

# ─── Application 入口 ───
-keep class com.mediahub.tv.MediaHubApp { *; }