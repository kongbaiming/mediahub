package com.mediahub.tv.ui.setup

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
import com.mediahub.tv.MediaHubApp
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.ui.MainActivity
import com.mediahub.tv.ui.theme.MediaHubTheme
import kotlinx.coroutines.launch

/**
 * 首次启动配置：让用户输入 NAS API 地址
 *
 * 触发条件：PreferencesRepository.apiBaseUrl == null
 * 完成后：写入 DataStore → 初始化 MediaHubApi → 跳转 MainActivity → finish() 自己
 *
 * TV 焦点说明：TextField + Button 的焦点是自动管理的；
 * 用户按 D-pad Center 在 TextField 中其实是选中文字（系统默认行为），
 * 按钮需要单独按一次 D-pad Center 才触发。已用 LaunchedEffect 自动聚焦。
 */
class SetupActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val app = application as MediaHubApp
        val prefs = app.prefs

        setContent {
            MediaHubTheme {
                SetupScreen(
                    onTest = { baseUrl -> MediaHubApi.init(baseUrl) },
                    onSaved = {
                        startActivity(MainActivity.intent(this))
                        finish()
                    },
                    initialValue = "http://10.0.0.1:3000",
                    onHealthCheck = { MediaHubApi.get().testConnection() },
                )
            }
        }
    }

    companion object {
        fun intent(context: Context): Intent =
            Intent(context, SetupActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
            }
    }
}

@Composable
fun SetupScreen(
    onTest: (String) -> Unit,
    onSaved: () -> Unit,
    onHealthCheck: suspend () -> Boolean,
    initialValue: String = "",
) {
    var input by remember { mutableStateOf(initialValue) }
    var testing by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val focusRequester = remember { FocusRequester() }
    val scope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        try {
            focusRequester.requestFocus()
        } catch (_: Throwable) { }
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(
                Brush.verticalGradient(
                    listOf(
                        androidx.compose.ui.graphics.Color(0xFF0F172A),
                        androidx.compose.ui.graphics.Color(0xFF1E1B4B),
                    ),
                ),
            )
            .padding(48.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp),
            modifier = Modifier.fillMaxWidth(0.6f),
        ) {
            Text(
                text = "MediaHub",
                color = androidx.compose.ui.graphics.Color.White,
                fontSize = 56.sp,
                fontWeight = FontWeight.Bold,
            )
            Text(
                text = "配置你的家庭媒资中心",
                color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                fontSize = 20.sp,
            )

            Spacer(modifier = Modifier.height(24.dp))

            OutlinedTextField(
                value = input,
                onValueChange = { input = it; error = null },
                label = { Text("NAS API 地址") },
                placeholder = { Text("http://192.168.1.100:3000") },
                singleLine = true,
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = androidx.compose.ui.graphics.Color(0xFF6366F1),
                    unfocusedBorderColor = androidx.compose.ui.graphics.Color(0xFF475569),
                    focusedTextColor = androidx.compose.ui.graphics.Color.White,
                    unfocusedTextColor = androidx.compose.ui.graphics.Color.White,
                    focusedLabelColor = androidx.compose.ui.graphics.Color(0xFF6366F1),
                    unfocusedLabelColor = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .focusRequester(focusRequester),
            )

            if (error != null) {
                Text(
                    text = error!!,
                    color = androidx.compose.ui.graphics.Color(0xFFEF4444),
                    fontSize = 14.sp,
                )
            }

            Button(
                onClick = {
                    if (testing) return@Button
                    val trimmed = input.trim().trimEnd('/')
                    when {
                        trimmed.isBlank() -> {
                            error = "请输入地址"
                            return@Button
                        }
                        !trimmed.startsWith("http://") && !trimmed.startsWith("https://") -> {
                            error = "必须以 http:// 或 https:// 开头"
                            return@Button
                        }
                    }
                    testing = true
                    error = null
                    onTest(trimmed)
                    scope.launch {
                        try {
                            val ok = onHealthCheck()
                            if (ok) {
                                onSaved()
                            } else {
                                error = "无法连接，请检查地址和服务状态"
                            }
                        } catch (t: Throwable) {
                            error = "错误：${t.message ?: "未知"}"
                        } finally {
                            testing = false
                        }
                    }
                },
                enabled = !testing,
                shape = RoundedCornerShape(8.dp),
                colors = ButtonDefaults.buttonColors(
                    containerColor = androidx.compose.ui.graphics.Color(0xFF6366F1),
                    contentColor = androidx.compose.ui.graphics.Color.White,
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
            ) {
                if (testing) {
                    CircularProgressIndicator(
                        color = androidx.compose.ui.graphics.Color.White,
                        modifier = Modifier.size(24.dp),
                        strokeWidth = 2.dp,
                    )
                } else {
                    Text("测试并继续", fontSize = 18.sp)
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "提示：地址可在 NAS Docker 控制台查看（默认 3000 端口）",
                color = androidx.compose.ui.graphics.Color(0xFF64748B),
                fontSize = 13.sp,
            )
        }
    }
}