package com.mediahub.tv.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color
import androidx.tv.material3.ExperimentalTvMaterial3Api
import androidx.tv.material3.TvMaterialTheme

/**
 * MediaHub TV 主题（深色为主，符合客厅观影场景）
 */
@OptIn(ExperimentalTvMaterial3Api::class)
@Composable
fun MediaHubTheme(content: @Composable () -> Unit) {
    TvMaterialTheme(
        colorScheme = androidx.tv.material3.darkColorScheme(
            primary = Color(0xFF6366F1),         // Indigo
            onPrimary = Color.White,
            secondary = Color(0xFFEC4899),       // Pink
            background = Color(0xFF0F172A),      // Slate-900
            surface = Color(0xFF1E293B),         // Slate-800
            onSurface = Color(0xFFE2E8F0),       // Slate-200
        ),
        content = content
    )
}
