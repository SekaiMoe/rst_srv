import gzip
from Crypto.Cipher import AES

key = b"从代码中找到的32字节或16字节Key"
iv = b"从代码中找到的16字节IV"

with open("masterdata_payload.enc", "rb") as f:
        cipher_data = f.read()

        cipher = AES.new(key, AES.MODE_CBC, iv)
        decrypted = cipher.decrypt(cipher_data)

        # 如果解密后是 Gzip 压缩包，解压即可得到 SQLite 数据库
        try:
              plain_db = gzip.decompress(decrypted)
        except:
              plain_db = decrypted

              with open("masterdata.db", "wb") as f:
                    f.write(plain_db)
                    
