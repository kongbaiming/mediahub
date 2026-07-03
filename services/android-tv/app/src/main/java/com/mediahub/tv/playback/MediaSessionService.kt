package com.mediahub.tv.playback

import android.app.PendingIntent
import android.content.Intent
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import com.mediahub.tv.ui.MainActivity

/**
 * 全局唯一的 ExoPlayer + MediaSession 持有者。
 *
 * 设计原则：
 *  - 一个进程内一个 ExoPlayer、一个 MediaSession
 *  - Service 常驻以支持后台播放 + 锁屏控制
 *  - PlaybackActivity 不再自己 new ExoPlayer，直接从 [MediaSessionManager] 拿
 */
class MediaSessionService : MediaSessionService() {

    private var mediaSession: MediaSession? = null

    override fun onCreate() {
        super.onCreate()
        val player = ExoPlayer.Builder(this)
            .setAudioAttributes(
                AudioAttributes.Builder()
                    .setUsage(C.USAGE_MEDIA)
                    .setContentType(C.AUDIO_CONTENT_TYPE_MOVIE)
                    .build(),
                /* handleAudioFocus = */ true,
            )
            .setHandleAudioBecomingNoisy(true)
            .build()
        mediaSession = MediaSession.Builder(this, player)
            .setSessionActivity(
                PendingIntent.getActivity(
                    this,
                    0,
                    Intent(this, MainActivity::class.java),
                    PendingIntent.FLAG_IMMUTABLE,
                ),
            )
            .build()
        MediaSessionManager.bind(player)
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? =
        mediaSession

    override fun onDestroy() {
        MediaSessionManager.unbind()
        mediaSession?.run {
            player.release()
            release()
            mediaSession = null
        }
        super.onDestroy()
    }
}

/**
 * 全局 player 句柄，由 [MediaSessionService.onCreate] 绑定、
 * [MediaSessionService.onDestroy] 解绑。
 *
 * PlaybackActivity 通过此对象拿到 ExoPlayer 并渲染 PlayerView，
 * 避免重复创建 player/session。
 */
object MediaSessionManager {
    @Volatile
    private var player: ExoPlayer? = null

    internal fun bind(p: ExoPlayer) { player = p }
    internal fun unbind() { player = null }

    fun get(): ExoPlayer =
        player ?: error("MediaSessionService not running. Start the service first.")
}
