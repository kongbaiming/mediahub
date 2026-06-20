package com.mediahub.tv.ui

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.lifecycle.lifecycleScope
import com.mediahub.tv.MediaHubApp
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.ui.browse.BrowseScreen
import com.mediahub.tv.ui.setup.SetupActivity
import com.mediahub.tv.ui.theme.MediaHubTheme
import kotlinx.coroutines.launch

/**
 * Main Activity - Leanback 入口
 *
 * 使用 Compose for TV 替代传统 Leanback Fragment（更现代的写法）
 *
 * 启动流程：
 *  1. onCreate() 启动一个协程读 DataStore
 *  2. 如果 apiBaseUrl 为空：跳到 SetupActivity，自己 finish()
 *  3. 如果非空：MediaHubApp.onCreate() 已 init MediaHubApi，进入 BrowseScreen
 */
class MainActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val app = application as MediaHubApp
        lifecycleScope.launch {
            val baseUrl = app.prefs.apiBaseUrl()
            if (baseUrl.isNullOrBlank()) {
                startActivity(SetupActivity.intent(this@MainActivity))
                finish()
                return@launch
            }
            // 设置 Compose 内容
            setContent {
                MediaHubTheme {
                    Surface(
                        modifier = Modifier.fillMaxSize(),
                        color = Color(0xFF0F172A)
                    ) {
                        BrowseScreen(api = MediaHubApi.get())
                    }
                }
            }
        }
    }

    companion object {
        fun intent(context: Context): Intent =
            Intent(context, MainActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            }
    }
}