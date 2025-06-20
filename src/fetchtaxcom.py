import json
import requests
from pathlib import Path
import pandas as pd
from urllib.parse import urlparse, parse_qs

import re
from urllib.parse import urlparse, parse_qs

def extract_raw_id(url_string):
    """
    Извлекает идентификатор (UUID-подобную строку) из URL.
    Поддерживает:
      - Параметр запроса id:   https://…?id=EB46FD18-6CB3-47FC-A37E-053E23B6BC0A
      - Последний сегмент пути: https://receipt.taxcom.ru/EB46FD18-6CB3-47FC-A37E-053E23B6BC0A
    """
    parsed = urlparse(url_string)

    segment = parsed.path.rstrip('/').split('/')[-1]
    # проверяем, что сегмент — только цифры, A–F и дефисы
    if re.fullmatch(r'[A-Fa-f0-9-]+', segment):
        return segment

    return None

def fetch_receipt(raw_id):
    #https://receipt.taxcom.ru/v01/show?id=A3185E9A-F5D3-4B46-9BB8-2ECBD2C034D4&nocat=True
    #https://receipt.taxcom.ru/Reciept/Upload/A3185E9A-F5D3-4B46-9BB8-2ECBD2C034D4
    url = f'https://receipt.taxcom.ru/Reciept/Upload/{raw_id}'
    response = requests.get(url, stream=True)
    if response.status_code != 200:
        return None
    content_type = response.headers.get('Content-Type', '')
    if 'application/pdf' not in content_type.lower():
        raise ValueError(f'Expected PDF, got {content_type!r}')
    
    # собрать всё в bytes
    return response.content
    

def save_receipt(pdf_data: bytes, raw_id: str, output_dir: str = 'taxcomePDF') -> Path:
    if not pdf_data:
        raise ValueError("Нет данных для сохранения")    

    out_folder = Path(output_dir)
    out_folder.mkdir(parents=True, exist_ok=True)

    file_path = out_folder / f'{raw_id}.pdf'
    with open(file_path, 'wb') as f:
        f.write(pdf_data)

    return file_path    

def process_csv(csv_path):
    # При чтении CSV файла
    df = pd.read_csv(csv_path)
    if 'readed' not in df.columns:
        df['readed'] = ''  # Добавляем столбец, если его нет

    for index, row in df.iterrows():
        print(f"Обработка строки {index}")
        if row['readed'] != 'readed':  # Проверяем, не был ли уже обработан этот запрос
            try:
                print(row.iloc[0])
                raw_id = extract_raw_id(row.iloc[0])
                print(raw_id)
                if raw_id:
                    pdf_data = fetch_receipt(raw_id)
                    save_receipt(pdf_data, raw_id)
                    
                    # Отмечаем как прочитанное
                    df.at[index, 'readed'] = 'readed'
                    df.to_csv(csv_path, index=False)
            except Exception as e:
                print(f"Ошибка при обработке строки {index}: {e}")

if __name__ == '__main__':
    # Укажите путь к вашему CSV файлу
    csv_file_path = 'inputtaxcom.csv'
    process_csv('./' + csv_file_path)
