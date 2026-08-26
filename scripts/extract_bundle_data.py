import UnityPy

def extract_bundle(file_path):
    print(f"\n[*] Processing {file_path}...")
    try:
        env = UnityPy.load(file_path)
    except Exception as e:
        print(f"[-] Load failed: {e}")
        return

    extracted_count = 0
    for obj in env.objects:
        type_str = getattr(obj.type, "name", str(obj.type))
        # 匹配 TextAsset 或未知二进制对象
        if "TextAsset" in type_str or obj.type == 49:
            try:
                data = obj.read()
                name = getattr(data, "name", getattr(data, "m_Name", f"export_{obj.path_id}"))
                raw_bytes = getattr(data, "script", getattr(data, "m_Script", getattr(data, "raw_data", None)))
                
                if raw_bytes is not None:
                    out_name = f"{name}.bin" if not name.endswith(".bin") else name
                    with open(out_name, "wb") as f:
                        f.write(bytes(raw_bytes))
                    print(f" [+] Extracted TextAsset: {out_name} ({len(raw_bytes)} bytes)")
                    extracted_count += 1
            except Exception as e:
                print(f" [-] Read object error: {e}")

    if extracted_count == 0:
        print(" [!] No TextAsset directly parsed. Listing all objects:")
        for obj in env.objects:
            print(f"   -> ID: {obj.path_id}, Type: {getattr(obj.type, 'name', str(obj.type))}")

extract_bundle("cri_audio_version_list")
extract_bundle("masterdata_encrypted")
