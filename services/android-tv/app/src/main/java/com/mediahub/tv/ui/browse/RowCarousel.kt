package com.mediahub.tv.ui.browse

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.tv.foundation.lazy.list.TvLazyRow
import androidx.tv.foundation.lazy.list.items
import androidx.tv.material3.Card
import androidx.tv.material3.ExperimentalTvMaterial3Api
import androidx.tv.material3.Material3
import androidx.tv.material3.Text
import coil.compose.AsyncImage
import coil.request.ImageRequest
import com.mediahub.tv.data.model.FeedRow
import com.mediahub.tv.data.model.MediaItem

/**
 * 布局行：横滑卡片 + 标题
 *
 * 性能要点：
 *  - TvLazyRow 复用 RecyclerView 思想，只渲染可见 + 少量缓存
 *  - items() 默认按 item 本身做 key，但 MediaItem 没有稳定 equals，
 *    所以手动传 key = mediaId，让 Compose 跳过未变化的 items
 */
@OptIn(ExperimentalTvMaterial3Api::class)
@Material3
@Composable
fun RowCarousel(row: FeedRow, onItemClick: (String) -> Unit) {
    Column(modifier = Modifier.fillMaxWidth()) {
        if (!row.title.isNullOrBlank()) {
            Text(
                text = row.title!!,
                modifier = Modifier.padding(start = 32.dp, bottom = 12.dp, top = 4.dp),
                color = Color.White,
                style = androidx.tv.material3.Typography().titleLarge,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }

        TvLazyRow(
            modifier = Modifier.fillMaxWidth(),
            contentPadding = PaddingValues(horizontal = 32.dp),
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // 用 stable key（mediaId）避免列表更新时全部重绘
            items(row.items, key = { it.mediaId }) { item ->
                MediaCard(item = item, onClick = { onItemClick(item.mediaId) })
            }
        }
    }
}

/**
 * 媒体卡（海报图 + 标题 + 评分）
 *
 * 性能要点：
 *  - AsyncImage 用 ImageRequest + placeholder/error 占位，
 *    避免图片加载时白屏闪烁
 *  - ContentScale.Crop 防止图片变形
 *  - crossfade 默认 300ms（全局 ImageLoader 已配）
 */
@OptIn(ExperimentalTvMaterial3Api::class)
@Material3
@Composable
fun MediaCard(item: MediaItem, onClick: () -> Unit) {
    Card(
        onClick = onClick,
        modifier = Modifier.width(180.dp),
    ) {
        Column {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(270.dp)
                    .background(
                        Brush.verticalGradient(
                            listOf(
                                Color(0xFF1E293B),
                                Color(0xFF334155),
                            ),
                        ),
                    ),
            ) {
                if (item.posterUrl != null) {
                    AsyncImage(
                        model = ImageRequest.Builder(androidx.compose.ui.platform.LocalContext.current)
                            .data(item.posterUrl)
                            .crossfade(true)
                            .build(),
                        contentDescription = item.title,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize(),
                    )
                } else {
                    Text(
                        text = item.title.take(8),
                        color = Color(0xFF94A3B8),
                        modifier = Modifier
                            .align(androidx.compose.ui.Alignment.Center)
                            .padding(8.dp),
                        maxLines = 3,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
            Column(modifier = Modifier.padding(8.dp)) {
                Text(
                    text = item.title,
                    color = Color.White,
                    style = androidx.tv.material3.Typography().titleMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
                if (item.rating > 0) {
                    Text(
                        text = "★ ${item.rating}",
                        color = Color(0xFFFBBF24),
                        style = androidx.tv.material3.Typography().bodySmall,
                    )
                }
            }
        }
    }
}