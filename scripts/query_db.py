import sqlite3

conn = sqlite3.connect("masterdata.db")
cursor = conn.cursor()

# 1. 查找所有包含 'path', 'url', 'sound', 'audio', 'asset', 'resource' 的表
cursor.execute("SELECT name FROM sqlite_master WHERE type='table';")
tables = [row[0] for row in cursor.fetchall()]

target_tables = [t for t in tables if any(k in t.lower() for k in ["sound", "audio", "bgm", "resource", "asset", "music", "file"])]
print(f"[*] 找到与资源相关的表: {target_tables}\n")

# 2. 打印这些表的结构与前 3 行数据
for t in target_tables:
    print(f"=== 表结构: {t} ===")
    cursor.execute(f"PRAGMA table_info({t});")
    cols = [col[1] for col in cursor.fetchall()]
    print(f"字段: {cols}")
    
    cursor.execute(f"SELECT * FROM {t} LIMIT 3;")
    rows = cursor.fetchall()
    for r in rows:
        print(r)
    print()
