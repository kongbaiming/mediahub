package com.mediahub.tv.ui.settings

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.tv.material3.ExperimentalTvMaterial3Api
import androidx.tv.material3.TvFocusable
import com.mediahub.tv.MediaHubApp
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.Profile
import com.mediahub.tv.ui.MainActivity
import com.mediahub.tv.ui.setup.SetupActivity
import com.mediahub.tv.ui.theme.MediaHubTheme
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.launch

/**
 * 设置页（设置 API URL + Profile 切换 + 退出登录 + 应用信息）
 *
 * 进入：MainActivity 右上角设置按钮
 * 退出：返回键
 *
 * Profile 切换是核心：家庭成员切换 → 影响 history / 收藏 / 续播
 */
class SettingsActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val app = application as MediaHubApp
        val prefs = app.prefs

        setContent {
            MediaHubTheme {
                SettingsScreen(
                    prefs = prefs,
                    api = MediaHubApi.get(),
                    onResetApi = {
                        // 重置 API：清空 baseUrl + token，回 SetupActivity
                        lifecycleScope.launch {
                            prefs.setApiBaseUrl("")
                            prefs.setToken(null)
                            startActivity(SetupActivity.intent(this@SettingsActivity))
                            finish()
                        }
                    },
                    onProfileChanged = {
                        // Profile 切换后强制刷新主页
                        startActivity(MainActivity.intent(this))
                        finish()
                    },
                )
            }
        }
    }

    companion object {
        fun intent(context: Context): Intent =
            Intent(context, SettingsActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
            }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun SettingsScreen(
    prefs: com.mediahub.tv.data.PreferencesRepository,
    api: MediaHubApi,
    onResetApi: () -> Unit,
    onProfileChanged: (Profile) -> Unit,
) {
    val scope = rememberCoroutineScope()
    var apiBaseUrl by remember { mutableStateOf("") }
    var profiles by remember { mutableStateOf<List<Profile>>(emptyList()) }
    var activeProfileId by remember { mutableStateOf<String?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        try {
            apiBaseUrl = prefs.apiBaseUrl() ?: ""
            activeProfileId = prefs.activeProfileId()
            val token = prefs.token()
            if (!token.isNullOrBlank()) {
                profiles = api.listProfiles(token)
            }
        } catch (t: Throwable) {
            error = t.message
        } finally {
            loading = false
        }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    listOf(
                        androidx.compose.ui.graphics.Color(0xFF0F172A),
                        androidx.compose.ui.graphics.Color(0xFF1E293B),
                    ),
                ),
            )
            .padding(48.dp),
    ) {
        Column(
            verticalArrangement = Arrangement.spacedBy(16.dp),
            modifier = Modifier.fillMaxWidth(0.7f),
        ) {
            // 标题
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = "设置",
                    color = androidx.compose.ui.graphics.Color.White,
                    fontSize = 40.sp,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = "← 返回",
                    color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                    fontSize = 14.sp,
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // ─── API 地址 ───
            SettingsSection(title = "服务器") {
                Text(
                    text = "API 地址：$apiBaseUrl",
                    color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                    fontSize = 16.sp,
                )
                Spacer(modifier = Modifier.height(8.dp))
                OutlinedButton(
                    onClick = onResetApi,
                    shape = RoundedCornerShape(8.dp),
                ) {
                    Text("重新配置")
                }
            }

            // ─── Profile 切换 ───
            SettingsSection(title = "家庭成员") {
                if (loading) {
                    CircularProgressIndicator(
                        color = androidx.compose.ui.graphics.Color(0xFF6366F1),
                        modifier = Modifier.size(32.dp),
                    )
                } else if (profiles.isEmpty()) {
                    Text(
                        text = "未登录或未找到成员。请在 Web Admin 添加。",
                        color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                        fontSize = 14.sp,
                    )
                } else {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        profiles.forEach { p ->
                            ProfileChip(
                                profile = p,
                                isActive = p.id == activeProfileId,
                                onClick = {
                                    scope.launch {
                                        prefs.setActiveProfileId(p.id)
                                        activeProfileId = p.id
                                        onProfileChanged(p)
                                    }
                                },
                            )
                        }
                    }
                }
            }

            // ─── 播放设置 ───
            SettingsSection(title = "播放") {
                Text(
                    text = "• 画质：自动（NAS 决定）",
                    color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                    fontSize = 14.sp,
                )
                Text(
                    text = "• 续播：开启（跨设备同步）",
                    color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                    fontSize = 14.sp,
                )
                Text(
                    text = "• 字幕：中文（默认）",
                    color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                    fontSize = 14.sp,
                )
            }

            // ─── 应用信息 ───
            SettingsSection(title = "关于") {
                Text(
                    text = "MediaHub v0.1.0",
                    color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                    fontSize = 14.sp,
                )
                Text(
                    text = "Apache License 2.0",
                    color = androidx.compose.ui.graphics.Color(0xFF64748B),
                    fontSize = 12.sp,
                )
            }

            if (error != null) {
                Text(
                    text = "错误：$error",
                    color = androidx.compose.ui.graphics.Color(0xFFEF4444),
                    fontSize = 14.sp,
                )
            }
        }
    }
}

@Composable
private fun SettingsSection(title: String, content: @Composable () -> Unit) {
    Surface(
        color = androidx.compose.ui.graphics.Color(0x991E293B),
        shape = RoundedCornerShape(12.dp),
        modifier = Modifier.fillMaxWidth(),
    ) {
        Column(modifier = Modifier.padding(20.dp)) {
            Text(
                text = title,
                color = androidx.compose.ui.graphics.Color(0xFF818CF8),
                fontSize = 14.sp,
                fontWeight = FontWeight.SemiBold,
            )
            Spacer(modifier = Modifier.height(12.dp))
            content()
        }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
private fun ProfileChip(
    profile: Profile,
    isActive: Boolean,
    onClick: () -> Unit,
) {
    val focusRequester = remember { FocusRequester() }
    val bg = when {
        isActive -> androidx.compose.ui.graphics.Color(0xFF6366F1)
        else -> androidx.compose.ui.graphics.Color(0xFF334155)
    }
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(20.dp),
        color = bg,
        modifier = Modifier
            .height(48.dp)
            .focusRequester(focusRequester),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.padding(horizontal = 20.dp),
        ) {
            Text(
                text = profile.avatarEmojiResolved.ifBlank { "👤" },
                fontSize = 20.sp,
            )
            Text(
                text = profile.name,
                color = androidx.compose.ui.graphics.Color.White,
                fontSize = 16.sp,
                fontWeight = if (isActive) FontWeight.Bold else FontWeight.Normal,
            )
            if (isActive) {
                Text(
                    text = "✓",
                    color = androidx.compose.ui.graphics.Color.White,
                    fontSize = 16.sp,
                )
            }
        }
    }
}