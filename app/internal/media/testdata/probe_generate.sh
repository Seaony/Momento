#!/bin/sh
# 重新生成 Probe 测试用的小视频样本。在本目录执行：sh probe_generate.sh
# 需要 ffmpeg（带 libx264、libx265、libvpx、libopus、libsvtav1）。
# 画面用 testsrc2，160x120、1～2 秒、高 CRF，让每个文件都在几十 KB 以内。
set -eu
cd "$(dirname "$0")"
FF="${FFMPEG:-ffmpeg}"
# 关掉 SVT-AV1 的启动日志。
export SVT_LOG=1
COMMON="-hide_banner -loglevel error -y"
# 输出选项：去掉源的元数据、写入可复现的字节。
OUT="-map_metadata -1 -fflags +bitexact"
SRC="-f lavfi -i testsrc2=size=160x120:rate=25"
TONE="-f lavfi -i sine=frequency=440:sample_rate=48000"

# H.264 MP4，没音频，moov 在文件尾（ffmpeg 默认不加 faststart）。2 秒 50 帧。
$FF $COMMON $SRC $OUT -t 2 -c:v libx264 -preset ultrafast -crf 40 -pix_fmt yuv420p -flags:v +bitexact probe_h264.mp4

# H.264 + AAC 音频，moov 挪到文件头。
$FF $COMMON $SRC $TONE $OUT -t 2 -map 0:v -map 1:a -c:v libx264 -preset ultrafast -crf 40 -pix_fmt yuv420p \
	-c:a aac -b:a 32k -flags:v +bitexact -flags:a +bitexact -movflags +faststart probe_h264_audio.mp4

# 带旋转元数据的 MOV：逆时针 -90 度，也就是播放时顺时针转 90 度（和 iPhone 竖拍一样）。
# 直接在转码时加旋转，ffmpeg 会把画面真的转过去；所以先编码，再用 -c copy 重封装只写元数据。
$FF $COMMON $SRC $OUT -t 1 -c:v libx264 -preset ultrafast -crf 40 -pix_fmt yuv420p -flags:v +bitexact probe_rot90_tmp.mov
$FF $COMMON -display_rotation:v:0 -90 -i probe_rot90_tmp.mov $OUT -c copy probe_rot90.mov
rm -f probe_rot90_tmp.mov

# HEVC 的 M4V，样本条目标成 hvc1（苹果设备的写法）；ffmpeg 的 m4v 默认封装不收 HEVC，所以指定 -f mp4。
$FF $COMMON $SRC $OUT -t 1 -c:v libx265 -preset ultrafast -crf 40 -pix_fmt yuv420p -tag:v hvc1 \
	-x265-params log-level=error -flags:v +bitexact -f mp4 probe_hevc.m4v

# AV1 MP4（样本条目 av01）。
$FF $COMMON $SRC $OUT -t 1 -c:v libsvtav1 -crf 55 -preset 12 -pix_fmt yuv420p -flags:v +bitexact probe_av1.mp4

# 碎片化 MP4：每个关键帧切一段，moov 里没有样本表，时长要从 moof 累加。
$FF $COMMON $SRC $OUT -t 2 -c:v libx264 -preset ultrafast -crf 40 -pix_fmt yuv420p -g 10 -flags:v +bitexact \
	-movflags frag_keyframe+empty_moov+default_base_moof probe_frag.mp4

# VP9 + Opus 的 WebM。
$FF $COMMON $SRC $TONE $OUT -t 1 -map 0:v -map 1:a -c:v libvpx-vp9 -crf 50 -b:v 0 -deadline realtime -cpu-used 8 \
	-c:a libopus -b:a 16k -flags:v +bitexact -flags:a +bitexact probe_vp9.webm

# VP8 的 WebM，没音频。
$FF $COMMON $SRC $OUT -t 1 -c:v libvpx -crf 50 -b:v 100k -deadline realtime -cpu-used 8 -flags:v +bitexact probe_vp8.webm

# 模拟实况照片视频那半：moov/meta 里写 com.apple.quicktime.content.identifier。
$FF $COMMON $SRC $OUT -t 1 -c:v libx264 -preset ultrafast -crf 40 -pix_fmt yuv420p -flags:v +bitexact \
	-metadata com.apple.quicktime.content.identifier=FD0B0708-C73F-4F6D-B57B-4169BB991D46 \
	-movflags use_metadata_tags probe_livephoto.mov

ls -l probe_*
