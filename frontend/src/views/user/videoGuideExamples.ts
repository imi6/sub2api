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
# 新建任务设置 API_KEY、VIDEO_MODEL；恢复已有任务改设 API_KEY、VIDEO_TASK_ID。
# 可选 API_BASE_URL（填写站点地址）；恢复模式只查询和下载，不会重新提交。
import os
import tempfile
import time
from contextlib import ExitStack
from pathlib import Path
from urllib.parse import quote, urlparse

import requests

BASE = os.environ.get("API_BASE_URL", ${JSON.stringify(baseUrl)}).rstrip("/")
if BASE.lower().endswith("/v1"):
    BASE = BASE[:-3]
HEADERS = {"Authorization": "Bearer " + os.environ["API_KEY"]}
output = Path("video.mp4")
if output.exists():
    raise RuntimeError("video.mp4 已存在，请另选文件名后再运行。")
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

task_id = os.environ.get("VIDEO_TASK_ID", "").strip()
if not task_id:
    # 只提交一次；网络异常不代表上游没有接受任务，切勿自动重发 POST。
    try:
        with requests.post(BASE + "/v1/videos", headers=HEADERS, json={
            "model": os.environ["VIDEO_MODEL"], "prompt": "一只猫沿着阳光下的海滩缓缓行走。",
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
print("任务 ID（请保存，用 VIDEO_TASK_ID 可恢复）：", task_id, flush=True)
task_url = BASE + "/v1/videos/" + quote(task_id, safe="")
deadline = time.monotonic() + 30 * 60

def wait(seconds):
    remaining = deadline - time.monotonic()
    if remaining > 0:
        time.sleep(min(seconds, remaining))


def retry_delay(response, fallback):
    try:
        return max(fallback, float(response.headers.get("Retry-After", "0")))
    except ValueError:
        return fallback


def check_quota(response):
    if response.status_code != 429:
        return
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
        raise RuntimeError(f"任务 {task_id} 被额度或订阅限制阻止，请处理后恢复：{message}")


def result_url(body):
    url = field(body, "url", "video.url", "content.video_url", "output.url", "video_url")
    try:
        parsed = urlparse(url) if isinstance(url, str) else None
    except ValueError:
        parsed = None
    if parsed is not None and parsed.scheme in {"http", "https"} and parsed.hostname:
        return url
    return None


def save_video(response):
    media_type = response.headers.get("Content-Type", "").split(";", 1)[0].strip().lower()
    if not (media_type.startswith("video/") or media_type == "application/octet-stream"):
        raise RuntimeError(f"下载并非视频文件，任务 {task_id}，Content-Type={media_type}")
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(mode="wb", dir=".", suffix=".mp4.part", delete=False) as file:
            temporary = Path(file.name)
            for chunk in response.iter_content(chunk_size=1024 * 1024):
                if time.monotonic() >= deadline:
                    raise TimeoutError(f"下载等待已达上限，请用 VIDEO_TASK_ID 恢复：{task_id}")
                if chunk:
                    file.write(chunk)
        if temporary.stat().st_size == 0:
            raise RuntimeError(f"下载文件为空，请用任务 ID 重试下载：{task_id}")
        temporary.replace(output)
    finally:
        if temporary is not None and temporary.exists():
            temporary.unlink()


# 只重试同一任务的查询和下载；生成 completed 之后仍可能等待上游归档。
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
        check_quota(response)
        if response.status_code == 429 or 500 <= response.status_code < 600:
            print(f"查询 HTTP {response.status_code}，稍后重试；任务 {task_id}")
            wait(retry_delay(response, backoff))
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
    if state in FAILURE:
        reason = field(result, "fail_reason", "error.message", "message", "error") or state
        raise RuntimeError(f"任务 {task_id} 已失败：{reason}")
    if state not in SUCCESS and state not in RUNNING:
        raise RuntimeError(f"无法识别状态结构，请保存响应与任务 ID 联系支持：{task_id}")
    if state not in SUCCESS:
        backoff = 15
        wait(15)
        continue

    try:
        # 先下载本站 /content；仅结果直链使用不带本站密钥的新请求。
        with ExitStack() as responses:
            download = responses.enter_context(requests.get(task_url + "/content", headers=HEADERS,
                    stream=True, timeout=(10, 90), allow_redirects=False))
            archive_pending = False
            if download.status_code == 409:
                # 只识别此明确错误，其他 409 不自动重试。兼容纯文本及 JSON 正文。
                archive_pending = "video archive is not ready" in download.text.lower()
            if download.status_code == 404 or archive_pending:
                url = result_url(result)
                delay = retry_delay(download, backoff)
                download.close()
                if url:
                    download = responses.enter_context(requests.get(url, stream=True,
                            timeout=(10, 90), allow_redirects=True))
                elif archive_pending:
                    print(f"生成已完成，上游视频归档尚未就绪；稍后查询并重试下载，任务 {task_id}")
                    wait(delay)
                    backoff = min(backoff * 2, 120)
                    continue
                else:
                    raise RuntimeError(f"下载端点不可用且没有有效结果链接，请联系支持：{task_id}")
            check_quota(download)
            if download.status_code == 429 or 500 <= download.status_code < 600:
                delay = retry_delay(download, backoff)
                print(f"下载 HTTP {download.status_code}，稍后重新查询并重试；任务 {task_id}")
                download.close()
                wait(delay)
                backoff = min(backoff * 2, 120)
                continue
            if not 200 <= download.status_code < 300:
                raise RuntimeError(f"下载失败，任务 {task_id}，HTTP {download.status_code}：{download.text[:500]}")
            save_video(download)
        break
    except TimeoutError:
        raise
    except requests.RequestException as error:
        print(f"下载网络异常，稍后重试同一任务 {task_id}：{error}")
        wait(backoff)
        backoff = min(backoff * 2, 120)
    except OSError as error:
        raise RuntimeError(f"文件未保存，请检查磁盘后用 VIDEO_TASK_ID 恢复：{task_id}") from error
else:
    raise TimeoutError(f"已达本地等待上限，任务未取消。设置 VIDEO_TASK_ID={task_id} 并使用原密钥可恢复；"
                       "若长期提示归档未就绪且无结果链接，请联系支持排查上游归档，不要重新生成。")
print("视频已保存：", output.resolve())`,
    error: JSON.stringify({
      error: { type: 'invalid_request_error', message: 'model is required' },
    }, null, 2),
  }
}
