#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Script kiểm tra số chữ của chương.
Kiểm tra số chữ của file chương, nhắc mở rộng khi thấp hơn 3000 chữ.

Cách đếm khớp với domain.WordCount trong ainovel-cli: tiếng Việt đếm theo âm tiết,
tiếng Trung đếm mỗi Hán tự là một chữ (một Hán tự ≈ một âm tiết, nên cùng ngưỡng
dùng được cho cả hai ngôn ngữ). Dấu câu và ký hiệu Markdown không được tính.
"""

import re
import sys
from pathlib import Path

# Sửa lỗi mã hoá trên console Windows
if sys.platform == 'win32':
    import io
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding='utf-8', errors='replace')

# Mỗi Hán tự là một chữ; mỗi cụm chữ/số liền nhau khác (âm tiết tiếng Việt) là một chữ.
WORD_RE = re.compile(r'[\u4e00-\u9fff]|[^\W_\u4e00-\u9fff]+')


def count_words(text: str) -> int:
    """Đếm số chữ (bỏ dấu câu và ký hiệu Markdown)."""
    text = re.sub(r'#{1,6}\s*', '', text)
    text = re.sub(r'\*\*(.*?)\*\*', r'\1', text)
    text = re.sub(r'\*(.*?)\*', r'\1', text)
    text = re.sub(r'~~(.*?)~~', r'\1', text)
    text = re.sub(r'`(.*?)`', r'\1', text)
    text = re.sub(r'\[(.*?)\]\(.*?\)', r'\1', text)
    return len(WORD_RE.findall(text))


def extract_content_from_chapter(file_path: Path) -> str:
    """Lấy phần chính văn của chương (bỏ dòng tiêu đề # đầu tiên)."""
    lines = file_path.read_text(encoding='utf-8').split('\n')
    for i, line in enumerate(lines):
        if not line.strip():
            continue
        if line.lstrip().startswith('#'):
            return '\n'.join(lines[i + 1:])
        break
    return '\n'.join(lines)


def check_chapter(file_path: str, min_words: int = 3000) -> dict:
    """Kiểm tra số chữ của một chương."""
    path = Path(file_path)
    if not path.exists():
        return {
            'file': str(path),
            'exists': False,
            'word_count': 0,
            'status': 'error',
            'message': f'Không tìm thấy file: {file_path}',
        }

    word_count = count_words(extract_content_from_chapter(path))
    status = 'pass' if word_count >= min_words else 'fail'
    message = f'Số chữ: {word_count}'
    if word_count >= min_words:
        message += ' (✓ đạt)'
    else:
        message += f' (✗ thiếu, cần ít nhất {min_words} chữ)'

    return {
        'file': str(path),
        'exists': True,
        'word_count': word_count,
        'status': status,
        'message': message,
    }


def check_all_chapters(directory: str, pattern: str = '*.md', min_words: int = 3000) -> list:
    """Kiểm tra mọi file chương khớp mẫu trong thư mục (mặc định chapters/NN.md)."""
    dir_path = Path(directory)
    if not dir_path.exists():
        print(f'Lỗi: thư mục không tồn tại - {directory}')
        return []

    chapter_files = sorted(dir_path.glob(pattern))
    return [check_chapter(str(chapter_file), min_words) for chapter_file in chapter_files]


def print_results(results: list, min_words: int = 3000) -> None:
    """In kết quả kiểm tra."""
    if not results:
        print('Không tìm thấy file chương nào')
        return

    total_words = 0
    passed = 0
    failed = 0

    print('\n' + '=' * 60)
    print('Báo cáo kiểm tra số chữ chương')
    print('=' * 60)

    for result in results:
        if not result['exists']:
            print(f'\n❌ {result["file"]}')
            print(f'   {result["message"]}')
            continue

        total_words += result['word_count']
        if result['status'] == 'pass':
            passed += 1
            icon = '✅'
        else:
            failed += 1
            icon = '⚠️ '

        print(f'\n{icon} {Path(result["file"]).name}')
        print(f'   {result["message"]}')

    print('\n' + '-' * 60)
    print(f'Tổng: {len(results)} chương | {passed} chương đạt | {failed} chương thiếu | Tổng số chữ: {total_words:,}')
    print('-' * 60)

    if failed > 0:
        print(f'\n⚠️  Có {failed} chương chưa đủ {min_words} chữ, gợi ý cách mở rộng:')
        print('   - Thêm miêu tả chi tiết (bối cảnh, tâm lý, hành động)')
        print('   - Thêm cảnh đối thoại')
        print('   - Mở rộng nội tâm nhân vật')
        print('   - Bổ sung câu chuyện nền')
        print('\n   Tham khảo: assets/references/content-expansion.md')


def main() -> None:
    if len(sys.argv) < 2:
        print('Cách dùng:')
        print('  Kiểm tra một chương:  python check_chapter_wordcount.py <đường dẫn file chương> [số chữ tối thiểu]')
        print('  Kiểm tra mọi chương: python check_chapter_wordcount.py --all <thư mục> [số chữ tối thiểu]')
        print('')
        print('Ví dụ:')
        print('  python check_chapter_wordcount.py output/novel/chapters/01.md')
        print('  python check_chapter_wordcount.py output/novel/chapters/01.md 3500')
        print('  python check_chapter_wordcount.py --all output/novel/chapters')
        print('  python check_chapter_wordcount.py --all output/novel/chapters 3500')
        return

    if sys.argv[1] == '--all':
        if len(sys.argv) < 3:
            print('Lỗi: dùng --all thì cần chỉ định thư mục')
            return
        directory = sys.argv[2]
        min_words = int(sys.argv[3]) if len(sys.argv) > 3 else 3000
        results = check_all_chapters(directory, min_words=min_words)
        print_results(results, min_words)
        return

    file_path = sys.argv[1]
    min_words = int(sys.argv[2]) if len(sys.argv) > 2 else 3000
    result = check_chapter(file_path, min_words)
    print_results([result], min_words)


if __name__ == '__main__':
    main()
