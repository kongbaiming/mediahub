package com.mediahub.tv.ui.playback

import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.ServiceConnection
import android.net.Uri
import android.os.Bundle
import android.os.IBinder
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.media3.common.MediaItem as ExoMediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.PlayerView
import com.mediahub.tv.data.PreferencesRepository
import com.mediahub.tv.data.api.MediaHubApi
import com.mediahub.tv.data.model.MediaDetail
import com.mediahub.tv.playback.MediaSessionManager
import com.mediahub.tv.playback.MediaSessionService
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

/**
 * 全屏播放 Activity
 *
 * 支持：
 *  - 整部播放（电影）：使用 media.storagePath
 *  - 单集播放（剧集）：使用 episode.filePath 或 media.storagePath + episodeId
 *  - 续播：从 resumeSec 秒处开始
 *  - 进度上报：携带 X-Profile-ID
 */
class PlaybackActivity : ComponentActivity() {

    private var bound = false
    private val connection = object : ServiceConnection {
        override fun onServiceConnected(name: ComponentName?, binder: IBinder?) {
            bound = true
        }
        override fun onServiceDisconnected(name: ComponentName?) {
            bound = false
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val mediaId = intent.getStringExtra(EXTRA_MEDIA_ID)
            ?: run { finish(); return }
        val episodeId = intent.getStringExtra(EXTRA_EPISODE_ID)
        val filePath = intent.getStringExtra(EXTRA_FILE_PATH)
        val resumeSec = intent.getIntExtra(EXTRA_RESUME_SEC, 0)

        // 启动并绑定到 MediaSessionService
        val serviceIntent = Intent(this, MediaSessionService::class.java)
        startService(serviceIntent)
        bindService(serviceIntent, connection, Context.BIND_AUTO_CREATE)

        setContent {
            PlaybackScreen(
                mediaId = mediaId,
                episodeId = episodeId,
                filePath = filePath,
                initialResumeSec = resumeSec,
                onClose = { finish() },
            )
        }
    }

    override fun onDestroy() {
        if (bound) {
            unbindService(connection)
            bound = false
        }
        super.onDestroy()
    }

    companion object {
        const val EXTRA_MEDIA_ID = "extra_media_id"
        const val EXTRA_EPISODE_ID = "extra_episode_id"
        const val EXTRA_FILE_PATH = "extra_file_path"
        const val EXTRA_RESUME_SEC = "extra_resume_sec"

        fun intent(
            context: Context,
            mediaId: String,
            episodeId: String? = null,
            filePath: String? = null,
            resumeSec: Int = 0,
        ): Intent =
            Intent(context, PlaybackActivity::class.java).apply {
                putExtra(EXTRA_MEDIA_ID, mediaId)
                episodeId?.let { putExtra(EXTRA_EPISODE_ID, it) }
                filePath?.let { putExtra(EXTRA_FILE_PATH, it) }
                putExtra(EXTRA_RESUME_SEC, resumeSec)
                flags = Intent.FLAG_ACTIVITY_NEW_TASK
            }
    }
}

@Composable
fun PlaybackScreen(
    mediaId: String,
    episodeId: String? = null,
    filePath: String? = null,
    initialResumeSec: Int = 0,
    onClose: () -> Unit,
) {
    var media by remember { mutableStateOf<MediaDetail?>(null) }
    var resumePosition by remember { mutableStateOf(initialResumeSec) }
    var playerError by remember { mutableStateOf<String?>(null) }

    val api = MediaHubApi.get()

    LaunchedEffect(mediaId) {
        val detail = withContext(Dispatchers.IO) { api.getMediaDetail(mediaId) }
        media = detail

        if (initialResumeSec == 0) {
            val resume = withContext(Dispatchers.IO) {
                runCatching { api.getResume(mediaId) }.getOrDefault(0)
            }
            resumePosition = resume
        }
    }

    Surface(modifier = Modifier.fillMaxSize(), color = Color.Black) {
        val current = media
        if (current == null) return@Surface

        PlayerViewComposable(
            detail = current,
            episodeId = episodeId,
            filePath = filePath,
            startPosition = resumePosition,
            onError = { playerError = it },
            onClose = onClose,
        )
    }
}

@Composable
private fun PlayerViewComposable(
    detail: MediaDetail,
    episodeId: String?,
    filePath: String?,
    startPosition: Int,
    onError: (String) -> Unit,
    onClose: () -> Unit,
) {
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var progressJob by remember { mutableStateOf<Job?>(null) }

    // 读取当前 Profile ID
    val prefs = remember { PreferencesRepository(context) }
    var profileId by remember { mutableStateOf("") }
    LaunchedEffect(Unit) {
        profileId = prefs.activeProfileId() ?: ""
    }

    val player: ExoPlayer = remember {
        runCatching { MediaSessionManager.get() }.getOrElse {
            ExoPlayer.Builder(context).build()
        }.also { p ->
            // 确定播放路径：优先用 episode 的 filePath，否则用 media 的 storagePath
            val playPath = filePath ?: detail.storagePath
            val hlsUrl = MediaHubApi.get().streamUrlHls(playPath, detail.id)
            val item = ExoMediaItem.Builder()
                .setUri(Uri.parse(hlsUrl))
                .setMediaId(detail.id)
                .setMediaMetadata(
                    MediaMetadata.Builder()
                        .setTitle(detail.title)
                        .setArtist(detail.genres.joinToString(", "))
                        .build(),
                )
                .build()
            p.setMediaItem(item)
            p.prepare()
            p.seekTo(startPosition * 1000L)
            p.playWhenReady = true
            p.addListener(object : androidx.media3.common.Player.Listener {
                override fun onPlayerError(error: androidx.media3.common.PlaybackException) {
                    onError(error.message ?: "Playback error")
                }
            })
        }
    }

    // 进度上报（携带 Profile ID）
    DisposableEffect(player) {
        progressJob = scope.launch {
            while (true) {
                delay(10_000)
                val pos = (player.currentPosition / 1000).toInt()
                val dur = (player.duration / 1000).toInt()
                if (dur > 0 && pos > 0) {
                    withContext(Dispatchers.IO) {
                        runCatching {
                            MediaHubApi.get().recordProgress(
                                profileId = profileId.ifEmpty { "tv-anonymous" },
                                mediaId = detail.id,
                                progress = pos,
                                duration = dur,
                            )
                        }
                    }
                }
            }
        }
        onDispose {
            progressJob?.cancel()
            progressJob = null
        }
    }

    AndroidView(
        factory = { ctx ->
            PlayerView(ctx).apply {
                this.player = player
                useController = true
            }
        },
        modifier = Modifier.fillMaxSize(),
    )
}
