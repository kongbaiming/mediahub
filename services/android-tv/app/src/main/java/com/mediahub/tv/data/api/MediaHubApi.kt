package com.mediahub.tv.data.api

import com.mediahub.tv.data.model.Feed
import com.mediahub.tv.data.model.MediaDetail
import com.mediahub.tv.data.model.Profile
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

/**
 * MediaHub API 客户端（OkHttp + kotlinx.serialization）
 */
class MediaHubApi(private val baseUrl: String) {

    private val client = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    private val json = Json {
        ignoreUnknownKeys = true
        coerceInputValues = true
    }

    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    /**
     * 拉取 Feed（播放端布局 + 数据）
     */
    suspend fun getFeed(platform: String = "android-tv", profileId: String? = null): Feed =
        withContext(Dispatchers.IO) {
            val url = buildString {
                append("$baseUrl/api/v1/feed/$platform")
                if (profileId != null) append("?profile_id=$profileId")
            }
            val req = Request.Builder()
                .url(url)
                .apply { profileId?.let { addHeader("X-Profile-ID", it) } }
                .build()

            client.newCall(req).execute().use { resp ->
                val body = resp.body?.string()
                    ?: throw MediaHubException("Empty response")
                if (!resp.isSuccessful) throw MediaHubException("HTTP ${resp.code}: $body")
                json.decodeFromString(Feed.serializer(), body)
            }
        }

    /**
     * 媒资详情
     */
    suspend fun getMediaDetail(id: String): MediaDetail = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url("$baseUrl/api/v1/media/$id")
            .build()
        client.newCall(req).execute().use { resp ->
            val body = resp.body?.string() ?: throw MediaHubException("Empty response")
            if (!resp.isSuccessful) throw MediaHubException("HTTP ${resp.code}")
            val wrapper = json.parseToJsonElement(body).jsonObject
            json.decodeFromJsonElement(MediaDetail.serializer(), wrapper["data"]!!)
        }
    }

    /**
     * Profile 列表
     */
    suspend fun listProfiles(token: String): List<Profile> = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url("$baseUrl/api/v1/profiles")
            .addHeader("Authorization", "Bearer $token")
            .build()
        client.newCall(req).execute().use { resp ->
            val body = resp.body?.string() ?: throw MediaHubException("Empty response")
            if (!resp.isSuccessful) throw MediaHubException("HTTP ${resp.code}")
            val wrapper = json.parseToJsonElement(body).jsonObject
            val data = wrapper["data"] ?: return@use emptyList<Profile>()
            json.decodeFromJsonElement(
                kotlinx.serialization.builtins.ListSerializer(Profile.serializer()),
                data
            )
        }
    }

    /**
     * 记录播放进度
     */
    suspend fun recordProgress(
        profileId: String,
        mediaId: String,
        progress: Int,
        duration: Int,
    ) = withContext(Dispatchers.IO) {
        val payload = """
            {"profile_id":"$profileId","media_id":"$mediaId",
             "progress":$progress,"duration":$duration,"device":"android-tv"}
        """.trimIndent()

        val req = Request.Builder()
            .url("$baseUrl/api/v1/history")
            .addHeader("X-Profile-ID", profileId)
            .post(payload.toRequestBody(jsonMediaType))
            .build()

        client.newCall(req).execute().use { /* ignore response */ }
    }

    /**
     * 获取续播位置
     */
    suspend fun getResume(mediaId: String): Int = withContext(Dispatchers.IO) {
        val req = Request.Builder()
            .url("$baseUrl/api/v1/resume/$mediaId")
            .build()
        client.newCall(req).execute().use { resp ->
            val body = resp.body?.string() ?: return@use 0
            if (!resp.isSuccessful) return@use 0
            val wrapper = json.parseToJsonElement(body).jsonObject
            val data = wrapper["data"] ?: return@use 0
            if (data.toString() == "null") return@use 0
            val obj = (data as kotlinx.serialization.json.JsonObject)
            obj["progress"]?.let { jsonPrimitive ->
                (jsonPrimitive as kotlinx.serialization.json.JsonPrimitive).content.toIntOrNull() ?: 0
            } ?: 0
        }
    }

    /**
     * 健康检查（用于 SetupActivity 测试连接）
     */
    suspend fun testConnection(): Boolean = withContext(Dispatchers.IO) {
        try {
            val req = Request.Builder().url("$baseUrl/healthz").build()
            client.newCall(req).execute().use { resp -> resp.isSuccessful }
        } catch (_: Throwable) {
            false
        }
    }

    /**
     * 搜索（关键词 → 媒资列表）
     *
     * 与服务端对齐：GET /api/v1/search?q=...&type=movie&limit=20
     * 返回：MediaItem 列表
     */
    suspend fun search(
        query: String,
        type: String? = null,
        limit: Int = 30,
    ): List<MediaItem> = withContext(Dispatchers.IO) {
        val url = buildString {
            append("$baseUrl/api/v1/search?q=")
            append(java.net.URLEncoder.encode(query, "UTF-8"))
            if (!type.isNullOrBlank()) append("&type=$type")
            append("&limit=$limit")
        }
        val req = Request.Builder().url(url).build()
        client.newCall(req).execute().use { resp ->
            val body = resp.body?.string() ?: throw MediaHubException("Empty response")
            if (!resp.isSuccessful) throw MediaHubException("HTTP ${resp.code}")
            val wrapper = json.parseToJsonElement(body).jsonObject
            val data = wrapper["data"] ?: return@use emptyList<MediaItem>()
            if (data.toString() == "null") return@use emptyList<MediaItem>()
            json.decodeFromJsonElement(
                kotlinx.serialization.builtins.ListSerializer(MediaItem.serializer()),
                data,
            )
        }
    }

    companion object {
        @Volatile
        private var instance: MediaHubApi? = null

        fun init(baseUrl: String) {
            instance = MediaHubApi(baseUrl)
        }

        fun get(): MediaHubApi =
            instance ?: error("MediaHubApi not initialized. Call init() first.")
    }
}

class MediaHubException(message: String) : Exception(message)

// JSON 辅助扩展
private val kotlinx.serialization.json.JsonElement.jsonObject
    get() = (this as kotlinx.serialization.json.JsonObject)

private val kotlinx.serialization.json.JsonObject.string: String
    get() = toString()
