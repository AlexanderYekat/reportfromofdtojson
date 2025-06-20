#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import shutil
import sys

def copy_files(list_file_path: str, src_dir: str, dst_dir: str) -> None:
    # Создаём целевую папку, если её ещё нет
    print(dst_dir)
    print(src_dir)
    print(list_file_path)
    os.makedirs(dst_dir, exist_ok=True)

    with open(list_file_path, 'r', encoding='utf-8') as f:
        for line in f:
            filename = line.strip()
            print(filename)
            if not filename:
                continue  # пропускаем пустые строки

            src_path = os.path.join(src_dir, filename)
            dst_path = os.path.join(dst_dir, filename)

            # Проверяем наличие исходного файла
            if not os.path.isfile(src_path):
                raise FileNotFoundError(f"Не найден файл: {src_path}")

            # Если в имени файла есть вложенные папки, создаём их в dst_dir
            os.makedirs(os.path.dirname(dst_path), exist_ok=True)

            # Копируем вместе с метаданными
            shutil.copy2(src_path, dst_path)
            print(f"Скопировано: {filename}")

    print("Копирование завершено успешно.")

if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(f"Использование: {sys.argv[0]} <список_файлов.txt> <папка_источник> <папка_назначения>")
        sys.exit(1)

    list_file, source_folder, destination_folder = sys.argv[1], sys.argv[2], sys.argv[3]
    try:
        copy_files(list_file, source_folder, destination_folder)
    except Exception as e:
        print(f"Ошибка: {e}", file=sys.stderr)
        sys.exit(2)
