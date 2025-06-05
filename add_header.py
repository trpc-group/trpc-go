import os

# 定义要添加的文本
HEADER_TEXT = """//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

"""

# 定义需要处理的文件扩展名
ALLOWED_EXTENSIONS = {'.go', '.proto'}

# 递归遍历目录及其子目录
def process_directory(directory):
    for root, _, files in os.walk(directory):
        for filename in files:
            # 检查文件扩展名是否符合要求
            if os.path.splitext(filename)[1] in ALLOWED_EXTENSIONS:
                file_path = os.path.join(root, filename)
                # 读取文件内容
                with open(file_path, 'r') as file:
                    content = file.read()
                
                # 检查文件开头是否已经包含 HEADER_TEXT
                if not content.startswith(HEADER_TEXT):
                    # 将 HEADER_TEXT 添加到文件开头
                    with open(file_path, 'w') as file:
                        file.write(HEADER_TEXT + content)
                    print(f"Header added to: {file_path}")
                else:
                    print(f"Header already exists in: {file_path}")

# 从当前目录开始处理
process_directory('.')
