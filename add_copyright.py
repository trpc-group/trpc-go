#!/usr/bin/env python3
import os
import re

COPYRIGHT_HEADER = '''//
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
'''

def has_copyright_header(content):
    # 检查文件是否已经包含版权声明
    return COPYRIGHT_HEADER.strip() in content

def add_copyright_header(file_path):
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        if has_copyright_header(content):
            print(f"File {file_path} already has copyright header")
            return False
        
        # 添加版权声明
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(COPYRIGHT_HEADER + content)
        
        print(f"Added copyright header to {file_path}")
        return True
    except Exception as e:
        print(f"Error processing {file_path}: {str(e)}")
        return False

def process_directory(directory):
    modified_files = 0
    for root, _, files in os.walk(directory):
        for file in files:
            if file.endswith('.go'):
                file_path = os.path.join(root, file)
                if add_copyright_header(file_path):
                    modified_files += 1
    
    return modified_files

if __name__ == '__main__':
    current_dir = os.getcwd()
    print(f"Processing directory: {current_dir}")
    modified = process_directory(current_dir)
    print(f"\nTotal files modified: {modified}") 