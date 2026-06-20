package com.mediahub.tv.playback

import android.content.Context
import androidx.media3.common.MediaMetadata
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService

/**
 * MediaSession Service（系统媒体控件集成）
 *
 * 关键作用：
 *  - 让用户能从电视主页或手机控制播放
 *  - 锁屏控制 / 通知栏控制
 *  - "正在播放" 卡片显示在系统设置里
 */
class MediaSessionService : MediaSessionService() {

    private var mediaSession: MediaSession? = null

    override fun onCreate() {
        super.onCreate()
        val player = ExoPlayer.Builder(this).build()
        mediaSession = MediaSession.Builder(this, player).build()
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? =
        mediaSession

    override fun onDestroy() {
        mediaSession?.run {
            player.release()
            release()
        }
        super.onDestroy()
    }
}

/**
 * MediaSession Manager 辅助（用于从 PlaybackActivity 调用）
 */
object MediaSessionManager {
    fun create(context: Context, player: ExoPlayer): MediaSession {
        return MediaSession.Builder(context, player).build().also {
            // 实际上 PlaybackActivity 用自己的 PlayerView，
            // MediaSessionService 才是常驻后台播放入口。
            // 这里只是占位实现。
        }
    }

    fun MediaSession.setMediaMetadata(title: String, artist: String?) {
        val metadata = MediaMetadata.Builder()
            .setTitle(title)
            .setArtist(artist ?: "")
            .build()
        player.mediaMetadata = metadata
    }
}
