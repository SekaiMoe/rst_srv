import UnityPy

def extract_direct():
    # 1. 提取 masterdata_encrypted 中的 TextAsset
    print("[*] Extracting masterdata_encrypted TextAsset...")
    env1 = UnityPy.load("masterdata_encrypted")
    for obj in env1.objects:
        if "TextAsset" in getattr(obj.type, "name", str(obj.type)) or obj.type == 49:
            try:
                # 优先获取原始字节，绕过字符串编码错误
                raw_bytes = obj.get_raw_data()
                # 如果 get_raw_data 返回完整对象流，按 Unity TextAsset 二进制结构提取:
                # [4字节 name_len] + [name] + [4字节 padding] + [4字节 script_len] + [script_bytes]
                with open("masterdata_raw_extracted.bin", "wb") as f:
                    f.write(raw_bytes)
                print(f" [+] Exported raw object stream: masterdata_raw_extracted.bin ({len(raw_bytes)} bytes)")
            except Exception as e:
                print(f" [-] Extract failed: {e}")

    # 2. 提取 cri_audio_version_list 中的 MonoBehaviour
    print("\n[*] Extracting cri_audio_version_list MonoBehaviour...")
    env2 = UnityPy.load("cri_audio_version_list")
    for obj in env2.objects:
        type_name = getattr(obj.type, "name", str(obj.type))
        if "MonoBehaviour" in type_name or obj.type == 114:
            try:
                raw_bytes = obj.get_raw_data()
                with open("cri_audio_mono.bin", "wb") as f:
                    f.write(raw_bytes)
                print(f" [+] Exported raw object stream: cri_audio_mono.bin ({len(raw_bytes)} bytes)")
            except Exception as e:
                print(f" [-] Extract failed: {e}")

extract_direct()
