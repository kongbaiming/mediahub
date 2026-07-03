package com.mediahub.tv.ui.detail

import android.content.Context
import android.content.Intent
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.mediahub.tv.data.PreferencesRepository
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.Episode
import com.mediahub.tv.data.model.MediaDetail
import com.mediahub.tv.data.model.MediaItem
import com.mediahub.tv.data.model.Season
import com.mediahub.tv.ui.playback.PlaybackActivity
import com.mediahub.tv.ui.theme.MediaHubTheme

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
                        startActivity(PlaybackActivity.intent(this, id, resumeSec = resumeSec))
                    },
                    onPlayEpisode = { mediaId, episodeId, filePath ->
                        startActivity(PlaybackActivity.intent(this, mediaId, episodeId = episodeId, filePath = filePath))
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
    onPlayEpisode: (mediaId: String, episodeId: String, filePath: String?) -> Unit,
    onBack: () -> Unit,
) {
    var detail by remember { mutableStateOf<MediaDetail?>(null) }
    var resumeSec by remember { mutableStateOf(0) }
    var selectedSeason by remember { mutableStateOf<Season?>(null) }
    var loading by remember { mutableStateOf(true) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(mediaId) {
        try {
            detail = api.getMediaDetail(mediaId)
            resumeSec = api.getResume(mediaId)
            // 自动选中第一季
            detail?.seasons?.firstOrNull()?.let { selectedSeason = it }
        } catch (t: Throwable) {
            error = t.message
        } finally {
            loading = false
        }
    }

    Box(modifier = Modifier.fillMaxSize().background(Color(0xFF0F172A))) {
        when {
            loading -> LoadingState()
            error != null -> ErrorState(error!!)
            detail != null -> ContentState(
                detail = detail!!,
                resumeSec = resumeSec,
                selectedSeason = selectedSeason,
                onSelectSeason = { selectedSeason = it },
                onPlay = { onPlay(mediaId, 0) },
                onResume = { onPlay(mediaId, resumeSec) },
                onPlayEpisode = { ep -> onPlayEpisode(mediaId, ep.id, ep.filePath) },
            )
        }
    }
}

@Composable
private fun LoadingState() {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator(color = Color(0xFF6366F1))
    }
}

@Composable
private fun ErrorState(message: String) {
    Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Text("加载失败", color = Color.White, fontSize = 24.sp, fontWeight = FontWeight.Bold)
            Spacer(modifier = Modifier.height(8.dp))
            Text(message, color = Color(0xFF94A3B8), fontSize = 14.sp)
        }
    }
}

@Composable
private fun ContentState(
    detail: MediaDetail,
    resumeSec: Int,
    selectedSeason: Season?,
    onSelectSeason: (Season) -> Unit,
    onPlay: () -> Unit,
    onResume: () -> Unit,
    onPlayEpisode: (Episode) -> Unit,
) {
    val showResume = resumeSec > 0

    LazyColumn(modifier = Modifier.fillMaxSize()) {
        // 背景 + 主信息区
        item {
            Box(modifier = Modifier.fillMaxWidth().height(480.dp)) {
                // 背景大图
                detail.backdropUrl?.let { url ->
                    AsyncImage(
                        model = url,
                        contentDescription = null,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize().background(
                            Brush.verticalGradient(
                                listOf(Color.Transparent, Color(0xCC0F172A), Color(0xFF0F172A)),
                            ),
                        ),
                    )
                }

                Column(modifier = Modifier.fillMaxSize().padding(48.dp)) {
                    // 标题
                    Text(
                        text = detail.title,
                        color = Color.White,
                        fontSize = 36.sp,
                        fontWeight = FontWeight.Bold,
                    )

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
                                    .background(Color(0xFF1E293B)),
                            )
                        }

                        // 右侧信息
                        Column(modifier = Modifier.fillMaxWidth()) {
                            // 元数据行
                            Row(
                                verticalAlignment = Alignment.CenterVertically,
                                horizontalArrangement = Arrangement.spacedBy(12.dp),
                            ) {
                                detail.year?.let { Text(it.toString(), color = Color(0xFFCBD5E1), fontSize = 16.sp) }
                                detail.runtime?.let { Text("${it} 分钟", color = Color(0xFFCBD5E1), fontSize = 16.sp) }
                                if (detail.rating > 0) {
                                    Text("★ ${detail.rating}", color = Color(0xFFFBBF24), fontSize = 16.sp)
                                }
                            }

                            Spacer(modifier = Modifier.height(8.dp))

                            // 类型标签
                            if (detail.genres.isNotEmpty()) {
                                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                                    detail.genres.take(4).forEach { g ->
                                        Surface(color = Color(0xFF1E293B), shape = RoundedCornerShape(6.dp)) {
                                            Text(g, color = Color(0xFFCBD5E1), fontSize = 12.sp,
                                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp))
                                        }
                                    }
                                }
                            }

                            Spacer(modifier = Modifier.height(20.dp))

                            // 操作按钮
                            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                                Button(
                                    onClick = if (detail.isSeries && selectedSeason?.episodes?.isNotEmpty() == true) {
                                        { onPlayEpisode(selectedSeason!!.episodes.first()) }
                                    } else onPlay,
                                    shape = RoundedCornerShape(8.dp),
                                    colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF6366F1)),
                                    modifier = Modifier.height(48.dp),
                                ) {
                                    Text(
                                        when {
                                            detail.isSeries -> "▶ 播放 S${selectedSeason?.seasonNumber ?: 1}E1"
                                            showResume -> "从头开始"
                                            else -> "▶ 播放"
                                        },
                                        fontSize = 16.sp,
                                    )
                                }
                                if (showResume) {
                                    OutlinedButton(
                                        onClick = onResume,
                                        shape = RoundedCornerShape(8.dp),
                                        modifier = Modifier.height(48.dp),
                                    ) {
                                        Text("续播 ${formatResume(resumeSec)}", color = Color.White, fontSize = 16.sp)
                                    }
                                }
                            }

                            Spacer(modifier = Modifier.height(20.dp))

                            // 简介
                            detail.overview?.let { ov ->
                                Text(ov, color = Color(0xFFCBD5E1), fontSize = 14.sp, lineHeight = 20.sp, maxLines = 4)
                            }

                            // 技术信息
                            Spacer(modifier = Modifier.height(16.dp))
                            Row(horizontalArrangement = Arrangement.spacedBy(16.dp)) {
                                detail.resolution?.let { TechChip("分辨率：$it") }
                                detail.videoCodec?.let { TechChip("视频：$it") }
                                detail.audioCodec?.let { TechChip("音频：$it") }
                                if (detail.hasSubtitle) TechChip("字幕 ✓")
                            }
                        }
                    }
                }
            }
        }

        // 季/集选择器（仅电视剧）
        if (detail.isSeries && detail.seasons != null && detail.seasons.isNotEmpty()) {
            item {
                Column(modifier = Modifier.padding(horizontal = 48.dp)) {
                    Spacer(modifier = Modifier.height(16.dp))
                    Text("选集", color = Color.White, fontSize = 20.sp, fontWeight = FontWeight.SemiBold)
                    Spacer(modifier = Modifier.height(12.dp))

                    // 季选择
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(detail.seasons!!) { season ->
                            val isSelected = season.id == selectedSeason?.id
                            FilterChip(
                                selected = isSelected,
                                onClick = { onSelectSeason(season) },
                                label = { Text("第 ${season.seasonNumber} 季") },
                                colors = FilterChipDefaults.filterChipColors(
                                    selectedContainerColor = Color(0xFF6366F1),
                                    selectedLabelColor = Color.White,
                                    containerColor = Color(0xFF1E293B),
                                    labelColor = Color(0xFFCBD5E1),
                                ),
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(12.dp))
                }
            }

            // 集列表
            val episodes = selectedSeason?.episodes ?: emptyList()
            items(episodes) { ep ->
                EpisodeRow(
                    episode = ep,
                    seasonNumber = selectedSeason?.seasonNumber ?: 1,
                    onClick = { onPlayEpisode(ep) },
                )
            }
        }

        // 底部留白
        item { Spacer(modifier = Modifier.height(48.dp)) }
    }
}

@Composable
private fun EpisodeRow(episode: Episode, seasonNumber: Int, onClick: () -> Unit) {
    Surface(
        onClick = onClick,
        color = Color.Transparent,
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 48.dp, vertical = 4.dp),
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .background(Color(0xFF1E293B), RoundedCornerShape(8.dp))
                .padding(16.dp),
        ) {
            // 集号
            Text(
                text = "E${episode.episodeNumber}",
                color = Color(0xFF6366F1),
                fontSize = 16.sp,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.width(48.dp),
            )

            // 缩略图
            episode.stillUrl?.let { url ->
                AsyncImage(
                    model = url,
                    contentDescription = null,
                    contentScale = ContentScale.Crop,
                    modifier = Modifier
                        .width(120.dp)
                        .height(68.dp)
                        .clip(RoundedCornerShape(6.dp))
                        .background(Color(0xFF334155)),
                )
                Spacer(modifier = Modifier.width(16.dp))
            }

            // 标题 + 时长
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = episode.title ?: "第 ${episode.episodeNumber} 集",
                    color = Color.White,
                    fontSize = 15.sp,
                    fontWeight = FontWeight.Medium,
                )
                if (episode.duration > 0) {
                    Text(
                        text = "${episode.duration / 60} 分钟",
                        color = Color(0xFF94A3B8),
                        fontSize = 12.sp,
                    )
                }
            }

            // 播放图标
            Text("▶", color = Color(0xFF6366F1), fontSize = 20.sp)
        }
    }
}

@Composable
private fun TechChip(text: String) {
    Surface(color = Color(0x331E293B), shape = RoundedCornerShape(4.dp)) {
        Text(text, color = Color(0xFF94A3B8), fontSize = 11.sp,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp))
    }
}

private fun formatResume(sec: Int): String {
    val h = sec / 3600
    val m = (sec % 3600) / 60
    return if (h > 0) "${h}h${m}m" else "${m}m"
}
