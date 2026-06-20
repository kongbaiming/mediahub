package com.mediahub.tv.ui.browse

import android.content.Context
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Settings
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.tv.foundation.lazy.list.TvLazyColumn
import androidx.tv.foundation.lazy.list.items
import androidx.tv.material3.ExperimentalTvMaterial3Api
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.FeedRow
import com.mediahub.tv.ui.detail.DetailActivity
import com.mediahub.tv.ui.search.SearchActivity
import com.mediahub.tv.ui.settings.SettingsActivity
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * BrowseScreen - 首页（拉布局 Feed + 渲染行卡）
 *
 * 顶部：标题 + 搜索图标 + 设置图标（图标按钮）
 * 主体：Feed 的每一行 → RowCarousel（横滑 + 卡片）
 *
 * 点击卡片：跳转到 DetailActivity
 */
@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun BrowseScreen(api: MediaHubApi = MediaHubApi.get()) {
    val context = LocalContext.current
    var rows by remember { mutableStateOf<List<FeedRow>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        loading = true
        try {
            val feed = withContext(Dispatchers.IO) {
                api.getFeed("android-tv")
            }
            rows = feed.rows
            error = null
        } catch (e: Exception) {
            error = e.message ?: "网络错误"
        } finally {
            loading = false
        }
    }

    Column(modifier = Modifier.fillMaxSize()) {
        // 顶部工具栏
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.SpaceBetween,
        ) {
            Text(
                text = "MediaHub",
                color = androidx.compose.ui.graphics.Color.White,
                style = androidx.compose.material3.MaterialTheme.typography.headlineSmall,
            )
            Row {
                IconButton(onClick = { context.startActivity(SearchActivity.intent(context)) }) {
                    Icon(
                        Icons.Default.Search,
                        contentDescription = "搜索",
                        tint = androidx.compose.ui.graphics.Color.White,
                    )
                }
                IconButton(onClick = { context.startActivity(SettingsActivity.intent(context)) }) {
                    Icon(
                        Icons.Default.Settings,
                        contentDescription = "设置",
                        tint = androidx.compose.ui.graphics.Color.White,
                    )
                }
            }
        }

        // 主体内容
        when {
            loading -> LoadingState()
            error != null -> ErrorState(error!!)
            rows.isEmpty() -> EmptyState()
            else -> {
                TvLazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(vertical = 16.dp),
                    verticalArrangement = Arrangement.spacedBy(20.dp)
                ) {
                    items(rows) { row ->
                        RowCarousel(
                            row = row,
                            onItemClick = { mediaId ->
                                context.startActivity(DetailActivity.intent(context, mediaId))
                            },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun LoadingState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = androidx.compose.ui.Alignment.Center,
    ) {
        androidx.compose.material3.CircularProgressIndicator(
            color = androidx.compose.ui.graphics.Color(0xFF6366F1),
        )
    }
}

@Composable
private fun ErrorState(message: String) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = androidx.compose.ui.Alignment.Center,
    ) {
        Column(horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally) {
            Text(
                "加载失败",
                color = androidx.compose.ui.graphics.Color.White,
                style = androidx.compose.material3.MaterialTheme.typography.headlineSmall,
            )
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                message,
                color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
            )
        }
    }
}

@Composable
private fun EmptyState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = androidx.compose.ui.Alignment.Center,
    ) {
        Text(
            "暂无内容",
            color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
            style = androidx.compose.material3.MaterialTheme.typography.headlineSmall,
        )
    }
}