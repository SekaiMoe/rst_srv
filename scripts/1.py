import UnityPy
import json

env = UnityPy.load("./Android_masterdata2")

for obj in env.objects:
    type_str = str(obj.type)
    if "AssetBundleManifest" in type_str or obj.type == 29013118:
        manifest = obj.read()
        print("=== 找到 AssetBundleManifest ===")
        
        try:
            raw_dict = manifest.to_dict()
            print(json.dumps(raw_dict, indent=2, ensure_ascii=False))
        except Exception as e:
            print(f"to_dict 转换异常: {e}")
            
        for attr in ["AssetBundleNames", "m_AssetBundleNames", "AssetBundleInfos", "m_AssetBundleInfos"]:
            if hasattr(manifest, attr):
                print(f"\n--- 字段: {attr} ---")
                print(getattr(manifest, attr))
