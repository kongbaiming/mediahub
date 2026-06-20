package com.mediahub.tv.ui.search

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.input.key.Key
import androidx.compose.ui.input.key.key
import androidx.compose.ui.input.key.onPreviewKeyEvent
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.MediaItem
import com.mediahub.tv.ui.detail.DetailActivity
import com.mediahub.tv.ui.theme.MediaHubTheme
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * 搜索页
 *
 * 交互：
 *  - 自动聚焦搜索框
 *  - 输入即搜（debounce 300ms）
 *  - 远程键盘 IME 支持 Enter 触发
 *  - 遥控器 OK 在搜索框中是 IME show/hide 切换
 *  - 结果 4 列网格，点击进入 DetailActivity
 */
class SearchActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MediaHubTheme {
                SearchScreen(
                    api = MediaHubApi.get(),
                    onResultClick = { mediaId ->
                        startActivity(DetailActivity.intent(this, mediaId))
                    },
                    onBack = { finish() },
                )
            }
        }
    }

    companion object {
        fun intent(context: Context): Intent =
            Intent(context, SearchActivity::class.java).apply {
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
            }
    }
}

@Composable
fun SearchScreen(
    api: MediaHubApi,
    onResultClick: (String) -> Unit,
    onBack: () -> Unit,
) {
    var query by remember { mutableStateOf("") }
    var results by remember { mutableStateOf<List<MediaItem>>(emptyList()) }
    var loading by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    val focusRequester = remember { FocusRequester() }
    val scope = rememberCoroutineScope()

    // 自动聚焦
    LaunchedEffect(Unit) {
        try {
            focusRequester.requestFocus()
        } catch (_: Throwable) { }
    }

    // Debounce 搜索
    LaunchedEffect(query) {
        if (query.isBlank()) {
            results = emptyList()
            error = null
            return@LaunchedEffect
        }
        delay(300)
        loading = true
        error = null
        try {
            results = api.search(query)
        } catch (t: Throwable) {
            error = t.message
            results = emptyList()
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
            .padding(32.dp)
            .onPreviewKeyEvent { keyEvent ->
                if (keyEvent.key == Key.Back) {
                    onBack()
                    true
                } else false
            },
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            // 顶部条
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = "搜索",
                    color = androidx.compose.ui.graphics.Color.White,
                    fontSize = 32.sp,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = "← 返回",
                    color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                    fontSize = 14.sp,
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            // 搜索框
            val keyboard = LocalSoftwareKeyboardController.current
            OutlinedTextField(
                value = query,
                onValueChange = { query = it },
                placeholder = { Text("输入电影 / 剧集名…", color = androidx.compose.ui.graphics.Color(0xFF64748B)) },
                singleLine = true,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Search),
                keyboardActions = KeyboardActions(
                    onSearch = {
                        keyboard?.hide()
                        // debounce 会自动触发，这里什么都不用做
                    },
                ),
                colors = OutlinedTextFieldDefaults.colors(
                    focusedBorderColor = androidx.compose.ui.graphics.Color(0xFF6366F1),
                    unfocusedBorderColor = androidx.compose.ui.graphics.Color(0xFF475569),
                    focusedTextColor = androidx.compose.ui.graphics.Color.White,
                    unfocusedTextColor = androidx.compose.ui.graphics.Color.White,
                ),
                modifier = Modifier
                    .fillMaxWidth()
                    .focusRequester(focusRequester),
            )

            Spacer(modifier = Modifier.height(24.dp))

            // 状态 / 结果区
            when {
                query.isBlank() -> EmptyHint()
                loading -> CenterMessage("搜索中…", showProgress = true)
                error != null -> CenterMessage("搜索失败：$error", showProgress = false)
                results.isEmpty() -> CenterMessage("未找到结果", showProgress = false)
                else -> ResultsGrid(results = results, onClick = onResultClick)
            }
        }
    }
}

@Composable
private fun EmptyHint() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        Text(
            text = "试试搜索「星际穿越」「三体」「流浪地球」",
            color = androidx.compose.ui.graphics.Color(0xFF64748B),
            fontSize = 16.sp,
        )
    }
}

@Composable
private fun CenterMessage(text: String, showProgress: Boolean) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center,
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            if (showProgress) {
                CircularProgressIndicator(color = androidx.compose.ui.graphics.Color(0xFF6366F1))
                Spacer(modifier = Modifier.height(12.dp))
            }
            Text(text, color = androidx.compose.ui.graphics.Color(0xFFCBD5E1), fontSize = 14.sp)
        }
    }
}

@Composable
private fun ResultsGrid(results: List<MediaItem>, onClick: (String) -> Unit) {
    LazyVerticalGrid(
        columns = GridCells.Fixed(4),
        horizontalArrangement = Arrangement.spacedBy(16.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
        contentPadding = PaddingValues(vertical = 8.dp),
    ) {
        items(results) { item ->
            ResultCard(item = item, onClick = { onClick(item.mediaId) })
        }
    }
}

@Composable
private fun ResultCard(item: MediaItem, onClick: () -> Unit) {
    val focusRequester = remember { FocusRequester() }
    Surface(
        onClick = onClick,
        shape = RoundedCornerShape(8.dp),
        color = androidx.compose.ui.graphics.Color(0xFF1E293B),
        modifier = Modifier
            .height(220.dp)
            .focusRequester(focusRequester),
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(160.dp)
                    .background(androidx.compose.ui.graphics.Color(0xFF334155)),
            ) {
                item.posterUrl?.let { url ->
                    AsyncImage(
                        model = url,
                        contentDescription = item.title,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize(),
                    )
                }
            }
            Column(modifier = Modifier.padding(8.dp)) {
                Text(
                    item.title,
                    color = androidx.compose.ui.graphics.Color.White,
                    fontSize = 13.sp,
                    maxLines = 1,
                    fontWeight = FontWeight.SemiBold,
                )
                if (item.year != null) {
                    Text(
                        item.year.toString(),
                        color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                        fontSize = 11.sp,
                    )
                }
            }
        }
    }
}