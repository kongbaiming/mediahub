package com.mediahub.tv.ui.detail

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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.MediaDetail
import com.mediahub.tv.data.model.MediaItem
import com.mediahub.tv.ui.playback.PlaybackActivity
import com.mediahub.tv.ui.theme.MediaHubTheme

/**
 * 媒资详情页
 *
 * 显示：
 *  - 背景大图（backdropUrl）
 *  - 海报 + 标题 + 元数据（年/类型/时长/评分）
 *  - 操作按钮：播放 / 续播（如果有 progress）/ 收藏
 *  - 简介
 *  - 相关推荐（横向滚动）
 *
 * 进入：BrowseScreen / SearchActivity 点卡片
 * 退出：返回键
 */
class DetailActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val mediaId = intent.getStringExtra(EXTRA_MEDIA_ID)
            ?: run { finish(); return }

        setContent {
            MediaHubTheme {
                DetailScreen(
                    mediaId = mediaId,
                    api = MediaHubApi.get(),
                    onPlay = { id, resumeSec ->
                        startActivity(PlaybackActivity.intent(this, id, resumeSec))
                    },
                    onBack = { finish() },
                )
            }
        }
    }

    companion object {
        const val EXTRA_MEDIA_ID = "extra_media_id"

        fun intent(context: Context, mediaId: String): Intent =
            Intent(context, DetailActivity::class.java).apply {
                putExtra(EXTRA_MEDIA_ID, mediaId)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
            }
    }
}

@Composable
fun DetailScreen(
    mediaId: String,
    api: MediaHubApi,
    onPlay: (mediaId: String, resumeSec: Int) -> Unit,
    onBack: () -> Unit,
) {
    var detail by remember { mutableStateOf<MediaDetail?>(null) }
    var resumeSec by remember { mutableStateOf(0) }
    var related by remember { mutableStateOf<List<MediaItem>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(mediaId) {
        try {
            detail = api.getMediaDetail(mediaId)
            resumeSec = api.getResume(mediaId)
            // 相关推荐（复用 Feed API 的 similar row 是当前实现的简化）
            // 实际可单独加 /api/v1/media/{id}/similar
            related = emptyList() // TODO: 等待后端 /similar 接口
        } catch (t: Throwable) {
            error = t.message
        } finally {
            loading = false
        }
    }

    Box(modifier = Modifier.fillMaxSize().background(androidx.compose.ui.graphics.Color(0xFF0F172A))) {
        when {
            loading -> LoadingState()
            error != null -> ErrorState(error!!)
            detail != null -> ContentState(
                detail = detail!!,
                resumeSec = resumeSec,
                related = related,
                onPlay = { onPlay(mediaId, 0) },
                onResume = { onPlay(mediaId, resumeSec) },
                onRelatedClick = { id -> /* TODO: 嵌套到 DetailActivity 自身 */ },
            )
        }
    }
}

@Composable
private fun LoadingState() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(color = androidx.compose.ui.graphics.Color(0xFF6366F1))
    }
}

@Composable
private fun ErrorState(message: String) {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text(
                text = "加载失败",
                color = androidx.compose.ui.graphics.Color.White,
                fontSize = 24.sp,
                fontWeight = FontWeight.Bold,
            )
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = message,
                color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                fontSize = 14.sp,
            )
        }
    }
}

@Composable
private fun ContentState(
    detail: MediaDetail,
    resumeSec: Int,
    related: List<MediaItem>,
    onPlay: () -> Unit,
    onResume: () -> Unit,
    onRelatedClick: (String) -> Unit,
) {
    val showResume = resumeSec > 0

    Box(modifier = Modifier.fillMaxSize()) {
        // 背景大图
        detail.backdropUrl?.let { url ->
            AsyncImage(
                model = url,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier
                    .fillMaxSize()
                    .background(
                        Brush.verticalGradient(
                            listOf(
                                androidx.compose.ui.graphics.Color.Transparent,
                                androidx.compose.ui.graphics.Color(0xCC0F172A),
                                androidx.compose.ui.graphics.Color(0xFF0F172A),
                            ),
                        ),
                    ),
            )
        }

        Column(modifier = Modifier.fillMaxSize().padding(48.dp)) {
            // 顶部条（标题 + 返回提示）
            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.SpaceBetween,
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(
                    text = detail.title,
                    color = androidx.compose.ui.graphics.Color.White,
                    fontSize = 36.sp,
                    fontWeight = FontWeight.Bold,
                )
                Text(
                    text = "← 返回",
                    color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
                    fontSize = 14.sp,
                )
            }

            Spacer(modifier = Modifier.height(24.dp))

            Row(
                horizontalArrangement = Arrangement.spacedBy(32.dp),
                modifier = Modifier.fillMaxWidth(),
            ) {
                // 海报
                detail.posterUrl?.let { url ->
                    AsyncImage(
                        model = url,
                        contentDescription = detail.title,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier
                            .width(220.dp)
                            .height(330.dp)
                            .clip(RoundedCornerShape(12.dp))
                            .background(androidx.compose.ui.graphics.Color(0xFF1E293B)),
                    )
                }

                // 右侧信息
                Column(modifier = Modifier.fillMaxWidth()) {
                    // 元数据行
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        if (detail.year != null) {
                            Text(
                                text = detail.year.toString(),
                                color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                                fontSize = 16.sp,
                            )
                        }
                        if (detail.runtime != null) {
                            Text(
                                text = "${detail.runtime} 分钟",
                                color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                                fontSize = 16.sp,
                            )
                        }
                        if (detail.rating > 0) {
                            Text(
                                text = "★ ${detail.rating}",
                                color = androidx.compose.ui.graphics.Color(0xFFFBBF24),
                                fontSize = 16.sp,
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    // 类型标签
                    if (detail.genres.isNotEmpty()) {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            detail.genres.take(4).forEach { g ->
                                Surface(
                                    color = androidx.compose.ui.graphics.Color(0xFF1E293B),
                                    shape = RoundedCornerShape(6.dp),
                                ) {
                                    Text(
                                        text = g,
                                        color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                                        fontSize = 12.sp,
                                        modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                                    )
                                }
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(20.dp))

                    // 操作按钮
                    Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                        Button(
                            onClick = onPlay,
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.buttonColors(
                                containerColor = androidx.compose.ui.graphics.Color(0xFF6366F1),
                            ),
                            modifier = Modifier.height(48.dp),
                        ) {
                            Text(if (showResume) "从头开始" else "▶ 播放", fontSize = 16.sp)
                        }
                        if (showResume) {
                            OutlinedButton(
                                onClick = onResume,
                                shape = RoundedCornerShape(8.dp),
                                modifier = Modifier.height(48.dp),
                            ) {
                                Text(
                                    "续播 ${formatResume(resumeSec)}",
                                    color = androidx.compose.ui.graphics.Color.White,
                                    fontSize = 16.sp,
                                )
                            }
                        }
                        OutlinedButton(
                            onClick = { /* TODO: 收藏 */ },
                            shape = RoundedCornerShape(8.dp),
                            modifier = Modifier.height(48.dp),
                        ) {
                            Text("☆ 收藏", color = androidx.compose.ui.graphics.Color.White, fontSize = 16.sp)
                        }
                    }

                    Spacer(modifier = Modifier.height(20.dp))

                    // 简介
                    detail.overview?.let { ov ->
                        Text(
                            text = ov,
                            color = androidx.compose.ui.graphics.Color(0xFFCBD5E1),
                            fontSize = 14.sp,
                            lineHeight = 20.sp,
                            maxLines = 4,
                        )
                    }

                    // 技术信息
                    Spacer(modifier = Modifier.height(16.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                        detail.resolution?.let {
                            TechChip("分辨率：$it")
                        }
                        detail.videoCodec?.let {
                            TechChip("视频：$it")
                        }
                        detail.audioCodec?.let {
                            TechChip("音频：$it")
                        }
                        if (detail.hasSubtitle) {
                            TechChip("字幕 ✓")
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(24.dp))

            // 相关推荐（如果有）
            if (related.isNotEmpty()) {
                Text(
                    text = "你可能也喜欢",
                    color = androidx.compose.ui.graphics.Color.White,
                    fontSize = 20.sp,
                    fontWeight = FontWeight.SemiBold,
                )
                Spacer(modifier = Modifier.height(12.dp))
                // 这里省略横向列表实现，复用 BrowseScreen 的 RowCarousel 思路
            }
        }
    }
}

@Composable
private fun TechChip(text: String) {
    Surface(
        color = androidx.compose.ui.graphics.Color(0x331E293B),
        shape = RoundedCornerShape(4.dp),
    ) {
        Text(
            text = text,
            color = androidx.compose.ui.graphics.Color(0xFF94A3B8),
            fontSize = 11.sp,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
        )
    }
}

private fun formatResume(sec: Int): String {
    val h = sec / 3600
    val m = (sec % 3600) / 60
    return if (h > 0) "${h}h${m}m" else "${m}m"
}