package com.mediahub.tv.ui.playback

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.media3.common.MediaItem as ExoMediaItem
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.PlayerView
import androidx.tv.material3.ExperimentalTvMaterial3Api
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.MediaDetail
import com.mediahub.tv.playback.MediaSessionManager
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

/**
 * 全屏播放 Activity
 */
class PlaybackActivity : ComponentActivity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val mediaId = intent.getStringExtra(EXTRA_MEDIA_ID)
            ?: run { finish(); return }
        val resumeSec = intent.getIntExtra(EXTRA_RESUME_SEC, 0)

        setContent {
            PlaybackScreen(
                mediaId = mediaId,
                initialResumeSec = resumeSec,
                onClose = { finish() }
            )
        }
    }

    companion object {
        const val EXTRA_MEDIA_ID = "extra_media_id"
        const val EXTRA_RESUME_SEC = "extra_resume_sec"

        fun intent(context: Context, mediaId: String, resumeSec: Int = 0): Intent =
            Intent(context, PlaybackActivity::class.java).apply {
                putExtra(EXTRA_MEDIA_ID, mediaId)
                putExtra(EXTRA_RESUME_SEC, resumeSec)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
            }
    }
}

@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun PlaybackScreen(
    mediaId: String,
    initialResumeSec: Int = 0,
    onClose: () -> Unit,
) {
    var media by remember { mutableStateOf<MediaDetail?>(null) }
    var resumePosition by remember { mutableStateOf(initialResumeSec) }

    val api = MediaHubApi.get()

    LaunchedEffect(mediaId) {
        // 1. 加载媒资
        val detail = withContext(Dispatchers.IO) { api.getMediaDetail(mediaId) }
        media = detail

        // 2. 如果没有外部传入续播位置，从 API 查询
        if (initialResumeSec == 0) {
            val resume = withContext(Dispatchers.IO) {
                try {
                    api.getResume(mediaId)
                } catch (e: Exception) {
                    0
                }
            }
            resumePosition = resume
        }
    }

    Surface(modifier = Modifier.fillMaxSize(), color = Color.Black) {
        media?.let { m ->
            PlayerViewComposable(
                detail = m,
                startPosition = resumePosition,
                onClose = onClose
            )
        }
    }
}

/**
 * ExoPlayer 集成
 */
@Composable
fun PlayerViewComposable(
    detail: MediaDetail,
    startPosition: Int,
    onClose: () -> Unit,
) {
    val context = androidx.compose.ui.platform.LocalContext.current
    val player = remember {
        ExoPlayer.Builder(context).build().apply {
            // 优先 HLS（弱网友好），fallback 直连
            val hlsUri = Uri.parse(
                "/api/v1/stream/hls?path=${java.net.URLEncoder.encode(detail.storagePath, "UTF-8")}" +
                "&media_id=${detail.id}"
            )
            val item = ExoMediaItem.Builder()
                .setUri(hlsUri)
                .setMediaId(detail.id)
                .build()
            setMediaItem(item)
            prepare()
            seekTo(startPosition * 1000L)
            playWhenReady = true
        }
    }

    // 进度上报
    LaunchedEffect(player) {
        val mediaSession = MediaSessionManager.create(context, player)
        mediaSession.setMediaMetadata(
            title = detail.title,
            artist = detail.genres.joinToString(", "),
        )

        // 周期性上报进度
        while (true) {
            kotlinx.coroutines.delay(10_000)
            val pos = (player.currentPosition / 1000).toInt()
            val dur = (player.duration / 1000).toInt()
            if (dur > 0 && pos > 0) {
                withContext(Dispatchers.IO) {
                    try {
                        MediaHubApi.get().recordProgress(
                            profileId = "tv-anonymous", // 简化为匿名 profile
                            mediaId = detail.id,
                            progress = pos,
                            duration = dur,
                        )
                    } catch (_: Exception) {}
                }
            }
        }
    }

    androidx.compose.runtime.DisposableEffect(player) {
        onDispose {
            player.release()
        }
    }

    // PlayerView（AndroidView 包装）
    androidx.compose.ui.viewinterop.AndroidView(
        factory = { ctx ->
            PlayerView(ctx).apply {
                this.player = player
                useController = true
            }
        },
        modifier = Modifier.fillMaxSize()
    )
}
