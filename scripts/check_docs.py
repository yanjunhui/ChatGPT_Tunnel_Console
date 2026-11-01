# Created by Yanjunhui
"""Check bilingual document links, code examples and immutable evidence values."""
import json
from pathlib import Path
import re


DOCS = [*sorted(Path('.').glob('*.md')), *sorted(Path('docs').rglob('*.md'))]


def blocks(text):
    return re.findall(r'^```([^\n]*)\n(.*?)^```\s*$', text, re.M | re.S)


def example(language, source):
    if language == 'json':
        return json.loads(source)
    if language == 'sh':
        return [line.split(' #', 1)[0].strip() for line in source.splitlines()
                if line.strip() and not line.lstrip().startswith('#')]
    if language == 'text' and 'ChatGPT / Codex' in source:
        return 'architecture diagram localized'
    return [' '.join(line.split()) for line in source.splitlines() if line.strip()]


def validate(path):
    text = path.read_text()
    lines = text.splitlines()
    assert lines[0] == '<!-- Created by Yanjunhui -->', path
    assert sum(bool(line.strip()) for line in lines) <= 999, path
    assert not any(line.rstrip() != line for line in lines), path
    assert len(re.findall(r'^```', text, re.M)) % 2 == 0, path
    for target in re.findall(r'\]\(([^)]+)\)', text):
        if '://' not in target and not target.startswith('#'):
            assert (path.parent / target.split('#')[0]).exists(), (path, target)
    return text


def compare(path, english):
    chinese_path = path.with_name(path.stem + '.zh-CN.md')
    chinese = chinese_path.read_text()
    assert '](' + chinese_path.name + ')' in english, path
    assert '](' + path.name + ')' in chinese, chinese_path
    en_blocks, zh_blocks = blocks(english), blocks(chinese)
    assert len(en_blocks) == len(zh_blocks), (path, 'code block count', len(en_blocks), len(zh_blocks))
    for index, ((en_lang, en_code), (zh_lang, zh_code)) in enumerate(zip(en_blocks, zh_blocks)):
        assert en_lang == zh_lang, (path, index, 'language')
        assert example(en_lang, en_code) == example(zh_lang, zh_code), (path, index, 'example differs')
    assert set(re.findall(r'\b[0-9a-f]{64}\b', english)) == set(re.findall(r'\b[0-9a-f]{64}\b', chinese)), path
    return {'english': str(path), 'chinese': str(chinese_path), 'matching_code_blocks': len(en_blocks)}


def main():
    pairs = []
    for path in DOCS:
        text = validate(path)
        if not path.name.endswith('.zh-CN.md'):
            pairs.append(compare(path, text))
    print(f"PASS: {len(DOCS)} documents; {len(pairs)} bilingual pairs; links, examples and hashes agree.")


if __name__ == '__main__':
    main()
