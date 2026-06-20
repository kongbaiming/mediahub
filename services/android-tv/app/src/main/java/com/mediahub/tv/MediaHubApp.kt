package com.mediahub.tv

import android.app.Application
import coil.ImageLoader
import coil.ImageLoaderFactory
import coil.disk.DiskCache
import coil.memory.MemoryCache
import com.mediahub.tv.data.PreferencesRepository
import com.mediahub.tv.data.api.MediaHubApi
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import okhttp3.OkHttpClient
import java.util.concurrent.TimeUnit

/**
 * Application 入口
 *
 * 职责：
 *  - 初始化 Preferences（DataStore）
 *  - 持有全局共享状态（API base URL、profile_id、token）
 *  - 异步从 DataStore 读取已保存的 API URL，初始化 MediaHubApi 单例
 *  - 配置 Coil 图片缓存（内存 25% / 磁盘 250MB / OkHttp 复用）
 */
class MediaHubApp : Application(), ImageLoaderFactory {

    val prefs by lazy { PreferencesRepository(this) }
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onCreate() {
        super.onCreate()
        instance = this

        // 异步读取已保存的 API URL，初始化 API 客户端
        scope.launch {
            val baseUrl = prefs.apiBaseUrl()
            if (!baseUrl.isNullOrBlank()) {
                MediaHubApi.init(baseUrl)
            }
        }
    }

    /**
     * Coil 全局 ImageLoader（性能优化点）
     *
     * 默认配置：内存缓存取系统 25%，磁盘缓存 250MB。
     * OkHttp 复用：避免每个图片请求都新建连接。
     *
     * 使用：Coil 的 AsyncImage 自动用全局 ImageLoader
     */
    override fun newImageLoader(): ImageLoader {
        return ImageLoader.Builder(this)
            .memoryCache {
                MemoryCache.Builder(this)
                    .maxSizePercent(0.25)  // 25% 系统可用内存
                    .build()
            }
            .diskCache {
                DiskCache.Builder()
                    .directory(cacheDir.resolve("image_cache"))
                    .maxSizeBytes(250L * 1024 * 1024)  // 250MB
                    .build()
            }
            .okHttpClient {
                OkHttpClient.Builder()
                    .connectTimeout(15, TimeUnit.SECONDS)
                    .readTimeout(30, TimeUnit.SECONDS)
                    .build()
            }
            .respectCacheHeaders(false)  // 优先用本地缓存
            .crossfade(true)
            .crossfade(300)
            .build()
    }

    companion object {
        @Volatile
        private var instance: MediaHubApp? = null

        fun get(): MediaHubApp =
            instance ?: error("MediaHubApp not initialized")
    }
}