export type VideoGuideExampleKey =
  | 'models'
  | 'create'
  | 'createResponse'
  | 'firstLast'
  | 'references'
  | 'multimodal'
  | 'polling'
  | 'pollingResponse'
  | 'download'
  | 'python'
  | 'error'

/** 只构建展示文本；示例中的密钥、模型、任务 ID 和素材地址都需要替换。 */
export function buildVideoGuideExamples(baseUrl: string): Record<VideoGuideExampleKey, string> {
  const apiUrl = `${baseUrl}/v1`
  const shellQuote = (value: string) => `'${value.replace(/'/g, `'"'"'`)}'`
  const authorization = `  -H 'Authorization: Bearer YOUR_API_KEY'`
  const request = (body: Record<string, unknown>) => [
    `curl --fail-with-body --silent --show-error ${shellQuote(`${apiUrl}/videos`)} \\`,
    `${authorization} \\`,
    `  -H 'Content-Type: application/json' \\`,
    `  --data ${shellQuote(JSON.stringify(body, null, 2))}`,
  ].join('\n')
  const common = {
    model: 'YOUR_VIDEO_MODEL',
    duration: 5,
    ratio: '16:9',
    resolution: '720p',
  }

  return {
    models: [
      `curl --fail-with-body --silent --show-error ${shellQuote(`${apiUrl}/models`)} \\`,
      authorization,
    ].join('\n'),
    create: request({ ...common, prompt: '一只猫沿着阳光下的海滩缓缓行走，背景是轻柔的海浪。' }),
    createResponse: JSON.stringify({ id: 'vid_EXAMPLE', status: 'queued' }, null, 2),
    firstLast: request({
      ...common,
      prompt: '从第一张图的构图平滑过渡到第二张图，保持人物外观一致。',
      first_frame: { image_url: 'https://example.com/first-frame.png' },
      last_frame: { image_url: 'https://example.com/last-frame.png' },
    }),
    references: request({
      ...common,
      prompt: '使用参考图中的人物与服装，生成转身微笑的镜头。',
      reference_images: [
        { image_url: 'https://example.com/reference-1.png' },
        { image_url: 'https://example.com/reference-2.png' },
      ],
    }),
    multimodal: request({
      ...common,
      prompt: '保持参考图中的人物外观，参考视频中的动作，并参考音频安排节奏。',
      reference_images: [{ image_url: 'https://example.com/reference.png' }],
      reference_videos: [{ video_url: 'https://example.com/reference.mp4' }],
      reference_audios: [{ audio_url: 'https://example.com/reference.mp3' }],
    }),
    polling: [
      `curl --fail-with-body --silent --show-error ${shellQuote(`${apiUrl}/videos/vid_EXAMPLE`)} \\`,
      authorization,
    ].join('\n'),
    pollingResponse: JSON.stringify({
      id: 'vid_EXAMPLE',
      status: 'completed',
      progress: 100,
      url: 'https://example.com/generated-video.mp4',
    }, null, 2),
    download: [
      `curl --fail --silent --show-error ${shellQuote(`${apiUrl}/videos/vid_EXAMPLE/content`)} \\`,
      `${authorization} \\`,
      `  -o video.mp4`,
    ].join('\n'),
    python: `# 安装依赖：pip install requests
# 运行前设置 API_KEY、VIDEO_MODEL；可选 API_BASE_URL（填写站点地址）。
import os
import tempfile
import time
from pathlib import Path
from urllib.parse import quote, urlparse

import requests

BASE = os.environ.get("API_BASE_URL", ${JSON.stringify(baseUrl)}).rstrip("/")
if BASE.lower().endswith("/v1"):
    BASE = BASE[:-3]
HEADERS = {"Authorization": "Bearer " + os.environ["API_KEY"]}
MODEL = os.environ["VIDEO_MODEL"]
output = Path("video.mp4")
if output.exists():
    raise RuntimeError("video.mp4 已存在，请另选文件名后再提交任务。")
SUCCESS = {"completed", "succeeded", "success", "done"}
FAILURE = {"failed", "error", "cancelled", "canceled", "expired"}
RUNNING = {"queued", "pending", "processing", "running", "in_progress", "submitted", "created"}

def field(body, *paths):
    # 兼容顶层字段及 data 对象；具体返回结构以实际渠道为准。
    sources = [body, body.get("data")] if isinstance(body, dict) else []
    for source in sources:
        for path in paths:
            value = source
            for part in path.split("."):
                value = value.get(part) if isinstance(value, dict) else None
            if value is not None and value != "":
                return value
    return None

# 只提交一次；网络异常不代表上游没有接受任务，切勿自动重发 POST。
try:
    with requests.post(BASE + "/v1/videos", headers=HEADERS, json={
        "model": MODEL, "prompt": "一只猫沿着阳光下的海滩缓缓行走。",
        "duration": 5, "ratio": "16:9", "resolution": "720p",
    }, timeout=(10, 90), allow_redirects=False) as response:
        if not 200 <= response.status_code < 300:
            raise RuntimeError(f"提交返回 HTTP {response.status_code}：{response.text[:500]}")
        created = response.json()
except (requests.RequestException, ValueError) as error:
    raise RuntimeError("未确认提交结果，请检查使用记录或联系支持；不要自动重新提交。") from error
task_id = field(created, "id", "task_id", "request_id")
if not isinstance(task_id, str) or not task_id.strip():
    raise RuntimeError("响应中没有可识别的任务 ID，请保存响应并联系支持，避免重复提交。")
print("任务 ID：", task_id)
task_url = BASE + "/v1/videos/" + quote(task_id, safe="")
deadline = time.monotonic() + 30 * 60

def wait(seconds):
    remaining = deadline - time.monotonic()
    if remaining > 0:
        time.sleep(min(seconds, remaining))

# 仅查询可重试；429、5xx 和网络异常使用独立退避。
backoff = 15
while time.monotonic() < deadline:
    try:
        response = requests.get(task_url, headers=HEADERS,
                                timeout=(10, 60), allow_redirects=False)
    except requests.RequestException as error:
        print(f"查询网络异常，稍后重试；任务 {task_id}：{error}")
        wait(backoff)
        backoff = min(backoff * 2, 120)
        continue
    with response:
        if response.status_code == 429:
            try:
                detail = response.json()
            except ValueError:
                detail = {}
            codes = " ".join(str(field(detail, path) or "").lower()
                             for path in ("error.code", "error.type", "code", "type"))
            message = str(field(detail, "error.message", "message", "error") or "")
            exhausted = ("quota_exceeded", "quota_exhausted", "insufficient_quota", "insufficient_balance",
                         "balance_exhausted", "subscription_expired", "subscription_limit_exceeded",
                         "api_key_rate_5h_exceeded", "api_key_rate_1d_exceeded", "api_key_rate_7d_exceeded")
            if any(marker in codes for marker in exhausted) or any(marker in message.lower() for marker in
                    ("余额不足", "余额已耗尽", "额度不足", "额度已用尽", "额度已耗尽", "配额已耗尽", "限额已用完",
                     "订阅已过期", "订阅已到期", "insufficient balance", "quota exhausted", "quota exceeded")):
                raise RuntimeError(f"任务 {task_id} 的查询被额度或订阅限制阻止，请处理后继续查询：{message}")
        if response.status_code == 429 or 500 <= response.status_code < 600:
            try:
                delay = max(backoff, float(response.headers.get("Retry-After", "0")))
            except ValueError:
                delay = backoff
            print(f"查询 HTTP {response.status_code}，稍后重试；任务 {task_id}")
            wait(delay)
            backoff = min(backoff * 2, 120)
            continue
        if not 200 <= response.status_code < 300:
            raise RuntimeError(f"查询失败，任务 {task_id}，HTTP {response.status_code}：{response.text[:500]}")
        try:
            result = response.json()
        except ValueError as error:
            raise RuntimeError(f"查询响应不是 JSON，请保存任务 ID 以便恢复：{task_id}") from error
    state = str(field(result, "status", "state") or "").lower()
    print("任务状态：", state or "未返回状态")
    if state in SUCCESS:
        break
    if state in FAILURE:
        reason = field(result, "fail_reason", "error.message", "message", "error") or state
        raise RuntimeError(f"任务 {task_id} 已失败：{reason}")
    if state not in RUNNING:
        raise RuntimeError(f"无法识别状态结构，请保存响应与任务 ID 联系支持：{task_id}")
    backoff = 15
    wait(15)
else:
    raise TimeoutError(f"已停止等待，任务并未被取消；稍后可用相同密钥继续查询：{task_id}")

# 先下载本站 /content；仅第三方结果链接使用无本站密钥的新请求。
temporary = None
try:
    download = requests.get(task_url + "/content", headers=HEADERS, stream=True,
                            timeout=(10, 90), allow_redirects=False)
    if download.status_code == 404:
        download.close()
        url = field(result, "url", "video.url", "content.video_url", "output.url", "video_url")
        try:
            parsed = urlparse(url) if isinstance(url, str) else None
        except ValueError:
            parsed = None
        if parsed is None or parsed.scheme not in {"http", "https"} or not parsed.hostname:
            raise RuntimeError(f"下载端点不可用且没有有效结果链接，请保存任务 ID 联系支持：{task_id}")
        # 独立请求不附带 HEADERS，结果域名及后续重定向都不会收到本站 API 密钥。
        download = requests.get(url, stream=True, timeout=(10, 90), allow_redirects=True)
    with download as response:
        if not 200 <= response.status_code < 300:
            raise RuntimeError(f"下载失败，任务 {task_id}，HTTP {response.status_code}；可单独重试下载。")
        media_type = response.headers.get("Content-Type", "").split(";", 1)[0].lower()
        if not (media_type.startswith("video/") or media_type == "application/octet-stream"):
            raise RuntimeError(f"下载并非视频文件，任务 {task_id}，Content-Type={media_type}")
        with tempfile.NamedTemporaryFile(mode="wb", dir=".", suffix=".mp4.part", delete=False) as file:
            temporary = Path(file.name)
            for chunk in response.iter_content(chunk_size=1024 * 1024):
                if chunk:
                    file.write(chunk)
        if temporary.stat().st_size == 0:
            raise RuntimeError(f"下载文件为空，请用任务 ID 重试下载：{task_id}")
        temporary.replace(output)
except (requests.RequestException, OSError) as error:
    raise RuntimeError(f"下载未完成，请用任务 ID 单独重试下载：{task_id}") from error
finally:
    if temporary is not None and temporary.exists():
        temporary.unlink()
print("视频已保存：", output.resolve())`,
    error: JSON.stringify({
      error: { type: 'invalid_request_error', message: 'model is required' },
    }, null, 2),
  }
}
