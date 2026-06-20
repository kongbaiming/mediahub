package com.mediahub.tv.data

import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map

private val Context.dataStore by preferencesDataStore(name = "mediahub_prefs")

/**
 * 用户偏好（DataStore）
 *
 * 存储：
 *  - API base URL
 *  - Token
 *  - Active Profile ID
 *
 * 两类 API：
 *  - 同步 snapshot 属性（apiBaseUrl / token / activeProfileId）：在 Activity.onCreate 启动时
 *    一次性读取，足够路由判断（是否首次启动 → SetupActivity）
 *  - Flow API（apiBaseUrlFlow / tokenFlow）：用于 UI 响应式订阅
 */
class PreferencesRepository(private val context: Context) {

    object Keys {
        val API_BASE_URL = stringPreferencesKey("api_base_url")
        val TOKEN = stringPreferencesKey("token")
        val ACTIVE_PROFILE_ID = stringPreferencesKey("active_profile_id")
    }

    // Flow API（响应式订阅用）
    val apiBaseUrlFlow: Flow<String?> = context.dataStore.data.map { it[Keys.API_BASE_URL] }
    val tokenFlow: Flow<String?> = context.dataStore.data.map { it[Keys.TOKEN] }
    val activeProfileIdFlow: Flow<String?> = context.dataStore.data.map { it[Keys.ACTIVE_PROFILE_ID] }

    // Suspend snapshot（启动路由判断用）
    suspend fun apiBaseUrl(): String? = apiBaseUrlFlow.first()
    suspend fun token(): String? = tokenFlow.first()
    suspend fun activeProfileId(): String? = activeProfileIdFlow.first()

    suspend fun setApiBaseUrl(url: String) {
        context.dataStore.edit { it[Keys.API_BASE_URL] = url }
    }

    suspend fun setToken(t: String?) {
        context.dataStore.edit {
            if (t == null) it.remove(Keys.TOKEN) else it[Keys.TOKEN] = t
        }
    }

    suspend fun setActiveProfileId(id: String) {
        context.dataStore.edit { it[Keys.ACTIVE_PROFILE_ID] = id }
    }
}