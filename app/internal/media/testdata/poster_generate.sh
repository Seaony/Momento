#!/bin/sh
# 生成 Poster 测试用的小视频样本。画面左半红、右半蓝，便于判断方向；
# 1 秒后左半变绿，用来确认确实按 at 取到了对应时刻的帧。
# 需要带 libx264 和 libvpx 的 ffmpeg（macOS: brew install ffmpeg）。
set -eu
cd "$(dirname "$0")"
FFMPEG=${FFMPEG:-ffmpeg}

# 左红右蓝，t>=1s 时左半变绿；10fps、每秒一个关键帧
SRC='color=c=red:s=160x180:r=10:d=2[a];color=c=blue:s=160x180:r=10:d=2[b];[a][b]hstack,drawbox=x=0:y=0:w=160:h=180:color=green:t=fill:enable=gte(t\,1)'

# H.264 MP4
"$FFMPEG" -y -v error -filter_complex "$SRC" -c:v libx264 -pix_fmt yuv420p -g 10 -crf 30 -movflags +faststart poster_h264.mp4

# 带 90° 旋转元数据的 MOV：编码画面仍是横的，播放时要顺时针转 90°（左半红转到上面）
"$FFMPEG" -y -v error -filter_complex "$SRC" -c:v libx264 -pix_fmt yuv420p -g 10 -crf 30 /tmp/poster_rot_src.mov
"$FFMPEG" -y -v error -display_rotation -90 -i /tmp/poster_rot_src.mov -c copy poster_rotate90.mov
rm -f /tmp/poster_rot_src.mov

# VP9 WebM
"$FFMPEG" -y -v error -filter_complex "$SRC" -c:v libvpx-vp9 -pix_fmt yuv420p -g 10 -b:v 100k poster_vp9.webm

# 只有 0.5 秒的短视频：0.25 秒后左半变绿，at 超出时长时应拿到第一帧（左红）而不是最后一帧
"$FFMPEG" -y -v error -filter_complex 'color=c=red:s=160x180:r=10:d=0.5[a];color=c=blue:s=160x180:r=10:d=0.5[b];[a][b]hstack,drawbox=x=0:y=0:w=160:h=180:color=green:t=fill:enable=gte(t\,0.25)' \
  -c:v libx264 -pix_fmt yuv420p -g 10 -crf 30 poster_short.mp4

# 乱码 .mp4：扩展名像视频，内容是纯文本
yes 'this is not a video' | head -c 4096 > poster_garbage.mp4
