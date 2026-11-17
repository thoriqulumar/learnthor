import json
from pathlib import Path

import argostranslate.package
import argostranslate.translate


ROOT = Path(__file__).resolve().parent
SOURCE = ROOT / "data" / "kanji-source"
OUTPUT = SOURCE / "translations-id.json"


def ensure_model():
    installed = argostranslate.translate.get_installed_languages()
    english = next((language for language in installed if language.code == "en"), None)
    indonesian = next((language for language in installed if language.code == "id"), None)
    if english and indonesian and english.get_translation(indonesian):
        return english.get_translation(indonesian)

    argostranslate.package.update_package_index()
    package = next(
        package
        for package in argostranslate.package.get_available_packages()
        if package.from_code == "en" and package.to_code == "id"
    )
    argostranslate.package.install_from_path(package.download())
    installed = argostranslate.translate.get_installed_languages()
    english = next(language for language in installed if language.code == "en")
    indonesian = next(language for language in installed if language.code == "id")
    return english.get_translation(indonesian)


def main():
    translator = ensure_model()
    translations = {}
    for level in ("n5", "n4", "n3"):
        entries = json.loads((SOURCE / f"{level}.json").read_text(encoding="utf-8"))
        for entry in entries:
            meaning = ", ".join(entry.get("meanings", []))
            translations[entry["character"]] = translator.translate(meaning).strip()
    OUTPUT.write_text(json.dumps(translations, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {len(translations)} Indonesian meanings to {OUTPUT}")


if __name__ == "__main__":
    main()
