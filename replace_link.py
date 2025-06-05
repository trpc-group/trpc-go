import os

# 定义要查找和替换的文本
OLD_TEXT = "\"trpc.group/trpc-go/trpc-go/\""
NEW_TEXT = "\"trpc.group/trpc-go/trpc-go\""

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
                
                # 替换文本
                if OLD_TEXT in content:
                    content = content.replace(OLD_TEXT, NEW_TEXT)
                    # 写回文件
                    with open(file_path, 'w') as file:
                        file.write(content)
                    print(f"Updated: {file_path}")
                else:
                    print(f"No changes needed: {file_path}")

# 从当前目录开始处理
process_directory('.')
