import csv
import json
import requests
import re
from pathlib import Path
import pandas as pd

def extract_raw_id(url_string):
    # Извлекаем RawId из строки вида '"=""https://ofd.ru/Document/RenderDoc?RawId=xxx"""'
    # Извлекаем RawId из строки вида '=ГИПЕРССЫЛКА("https://ofd.ru/Document/RenderDoc?RawId=671115e6-1b3c-5e94-bac3-ecab745d390b";"Перейти")'
    match = re.search(r'RawId=([a-f0-9-]+)', url_string)
    if match:
        return match.group(1)
    return None

def fetch_receipt(raw_id):
    url = f'https://ofd.ru/Document/ReceiptJsonDownload?DocId={raw_id}'
    response = requests.get(url)
    if response.status_code == 200:
        return response.json()
    return None

def save_receipt(receipt_data):
    if not receipt_data:
        return
    
    # Получаем необходимые значения из JSON
    fn_number = receipt_data['Document']['FN_FactoryNumber']
    receipt_number = receipt_data['DocNumber']
    
    # Создаем директорию для сохранения файлов, если её нет
    output_dir = Path('receipts')
    output_dir.mkdir(exist_ok=True)
    
    # Формируем имя файла и сохраняем JSON
    filename = output_dir / f'{fn_number}_{receipt_number}.json'
    with open(filename, 'w', encoding='utf-8') as f:
        json.dump(receipt_data, f, ensure_ascii=False, indent=2)

def process_csv(csv_path):
    # При чтении CSV файла
    df = pd.read_csv(csv_path)
    if 'readed' not in df.columns:
        df['readed'] = ''  # Добавляем столбец, если его нет

    for index, row in df.iterrows():
        if row['readed'] != 'readed':  # Проверяем, не был ли уже обработан этот запрос
            try:
                raw_id = extract_raw_id(row.iloc[0])
                if raw_id:
                    receipt_data = fetch_receipt(raw_id)
                    save_receipt(receipt_data)
                    
                    # Отмечаем как прочитанное
                    df.at[index, 'readed'] = 'readed'
                    df.to_csv(csv_path, index=False)
            except Exception as e:
                print(f"Ошибка при обработке строки {index}: {e}")

if __name__ == '__main__':
    # Укажите путь к вашему CSV файлу
    csv_file_path = 'input.csv'
    process_csv('./' + csv_file_path)
