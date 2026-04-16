#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import base64
import mimetypes
import os
import requests
import time
import hashlib
import urllib.parse
import sys


DEFAULT_USER_AGENT = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
    "AppleWebKit/537.36 (KHTML, like Gecko) "
    "Chrome/140.0.0.0 Safari/537.36"
)


def generate_token(e: str, timestamp: str) -> str:
    s = hashlib.md5(e.encode("utf-8")).hexdigest()
    combined = s + "pic_edit" + timestamp
    final_hash = hashlib.md5(combined.encode("utf-8")).hexdigest()
    return final_hash[:5]


def image_to_base64(file_path: str) -> str:
    with open(file_path, "rb") as image_file:
        image_data = image_file.read()
        return base64.b64encode(image_data).decode("utf-8")


def image_url_to_base64(image_url: str) -> tuple[str, str]:
    response = requests.get(image_url, timeout=10, headers={"User-Agent": DEFAULT_USER_AGENT})
    response.raise_for_status()

    content_type = response.headers.get("content-type", "").split(";")[0].strip()
    if not content_type or "image" not in content_type:
        content_type = "image/jpeg"

    base64_str = base64.b64encode(response.content).decode("utf-8")
    return content_type, base64_str


def image_to_base64_data_url(image_path: str) -> str:
    if not os.path.exists(image_path):
        raise FileNotFoundError(f"文件不存在: {image_path}")

    mime_type, _ = mimetypes.guess_type(image_path)
    if mime_type is None:
        raise ValueError("无法识别图片类型，请确认文件格式")

    base64_str = image_to_base64(image_path)
    return f"data:{mime_type};base64,{base64_str}"


def url_to_base64_data_url(image_url: str) -> str:
    mime_type, base64_str = image_url_to_base64(image_url)
    return f"data:{mime_type};base64,{base64_str}"


def upload_to_baidu(image_source: str, is_url: bool = False) -> str:
    url = "https://image.baidu.com/aigc/pic_upload"
    headers = {
        "Accept": "*/*",
        "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
        "Cache-Control": "no-cache",
        "Connection": "keep-alive",
        "Origin": "https://image.baidu.com",
        "Pragma": "no-cache",
        "Referer": "https://image.baidu.com/",
        "Sec-Fetch-Dest": "empty",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Site": "same-origin",
        "User-Agent": DEFAULT_USER_AGENT,
        "sec-ch-ua": '"Chromium";v="140", "Not=A?Brand";v="24", "Google Chrome";v="140"',
        "sec-ch-ua-mobile": "?0",
        "sec-ch-ua-platform": '"Windows"',
        "Content-Type": "application/x-www-form-urlencoded;charset=UTF-8"
    }

    timestamp = str(int(time.time() * 1000))

    if is_url:
        mime_type, base64_string = image_url_to_base64(image_source)
    else:
        mime_type, _ = mimetypes.guess_type(image_source)
        if mime_type is None:
            mime_type = "image/jpeg"
        base64_string = image_to_base64(image_source)

    e = f"data:{mime_type};base64,{base64_string}"
    token = generate_token(e, timestamp)

    payload = {
        "token": token,
        "scene": "pic_edit",
        "picInfo": e,
        "timestamp": timestamp,
    }
    payload = urllib.parse.urlencode(payload)

    response = requests.post(url, headers=headers, data=payload, timeout=30)
    response.raise_for_status()

    result = response.json()
    if "data" not in result or "url" not in result["data"]:
        raise RuntimeError(f"百度返回异常: {result}")

    return result["data"]["url"]


def print_help():
    print("用法:")
    print("  python v2.py dataurl-file <本地图片路径>")
    print("  python v2.py dataurl-url <网络图片地址>")
    print("  python v2.py upload-file <本地图片路径>")
    print("  python v2.py upload-url <网络图片地址>")
    print("")
    print("说明:")
    print("  dataurl-file  本地图片转 data:mimetype;base64,...")
    print("  dataurl-url   网络图片转 data:mimetype;base64,...")
    print("  upload-file   本地图片上传百度，返回百度图片地址")
    print("  upload-url    网络图片上传百度，返回百度图片地址")
    print("")
    print("示例:")
    print("  python v2.py dataurl-file ./test.png")
    print('  python v2.py dataurl-url "https://gips3.baidu.com/it/u=3886271102,3123389489&fm=3028&app=3028&f=JPEG&fmt=auto?w=1280&h=960"')
    print("  python v2.py upload-file ./test.png")
    print('  python v2.py upload-url "https://ts3.tc.mm.bing.net/th?id=ORMS.8d6f0c90cd5bec42f9c73e3d207913ac&pid=Wdp&w=268&h=140&qlt=90&c=1&rs=1&dpr=1&p=0"')


def main():
    if len(sys.argv) < 2:
        print_help()
        sys.exit(1)

    cmd = sys.argv[1]

    if cmd in ("-h", "--help", "help"):
        print_help()
        sys.exit(0)

    if len(sys.argv) < 3:
        print("参数不足")
        print_help()
        sys.exit(1)

    arg = sys.argv[2]

    try:
        if cmd == "dataurl-file":
            print(image_to_base64_data_url(arg))

        elif cmd == "dataurl-url":
            print(url_to_base64_data_url(arg))

        elif cmd == "upload-file":
            print(upload_to_baidu(arg, is_url=False))

        elif cmd == "upload-url":
            print(upload_to_baidu(arg, is_url=True))

        else:
            print(f"未知命令: {cmd}")
            print_help()
            sys.exit(1)

    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(2)


if __name__ == "__main__":
    main()