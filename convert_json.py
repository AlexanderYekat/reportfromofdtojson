import json
import os
from datetime import datetime
import glob
import csv

def find_fpd_value(fn_number, fd_number, csv_path='./input_json/checks.csv'):
    """Ищет значение ФПД в CSV файле по номеру ФН и номеру ФД"""
    try:
        with open(csv_path, 'r', encoding='utf-8') as f:
            reader = csv.DictReader(f, delimiter=';')
            for row in reader:
                if (str(row['Заводской номер ФН']) == str(fn_number) and 
                    str(row['Номер ФД']) == str(fd_number)):
                    return row.get('ФПД', '')
    except Exception as e:
        print(f"Ошибка при чтении CSV файла: {str(e)}")
    return ''

def convert_json(input_json):
    """Конвертирует входной JSON в требуемый формат"""
    
    # Проверяем успешность входного JSON
    if input_json.get("Status") != "Success":
        return None
        
    tlv = input_json["Data"]["TlvDictionary"]
    
    # Получаем номер ФН и номер ФД
    fn_number = tlv.get("1041", "")
    fd_number = tlv.get("1040", "")
    
    # Ищем значение ФПД
    fpd_value = find_fpd_value(fn_number, fd_number)
    
    # Создаем базовую структуру выходного JSON
    output = {
        "type": "sellCorrection", # По умолчанию чек коррекции прихода
        "electronically": True,
        "taxationType": "",
        "correctionType": "self",
        "correctionBaseDate": "",
        "operator": {
            "name": tlv.get("1021", ""),  # Имя кассира
        },
        "items": [],
        "payments": []
    }
    
    # Определяем тип чека на основе тега 1054
    operation_type = tlv.get("1054")
    if operation_type == 1:
        output["type"] = "sellCorrection"
    elif operation_type == 2:
        output["type"] = "sellReturnCorrection" 
    elif operation_type == 3:
        output["type"] = "buyCorrection" 
    elif operation_type == 4:
        output["type"] = "buyReturnCorrection"

    taxation_type = tlv.get("1055", 0)
    if taxation_type & (1 << 0):  # Проверяем 0-й бит
        output["taxationType"] = "osn"
    elif taxation_type & (1 << 1):  # Проверяем 1-й бит
        output["taxationType"] = "usnIncome"
    elif taxation_type & (1 << 2):  # Проверяем 2-й бит
        output["taxationType"] = "usnIncomeOutcome"
    elif taxation_type & (1 << 3):  # Проверяем 3-й бит
        output["taxationType"] = "envd"
    elif taxation_type & (1 << 4):  # Проверяем 4-й бит
        output["taxationType"] = "esn"
    elif taxation_type & (1 << 5):  # Проверяем 5-й бит
        output["taxationType"] = "patent"

    # Добавляем дату коррекции
    if "1012" in tlv:
        try:
            date = datetime.strptime(tlv["1012"], "%Y-%m-%dT%H:%M:%S")
            output["correctionBaseDate"] = date.strftime("%Y.%m.%d")
        except:
            output["correctionBaseDate"] = ""

    if "1040" in tlv:
        userAttribute = {
            "type": "userAttribute",
            "name": "ФД",
            "value": str(tlv["1040"]),
            "print": True
        }
        output["items"].append(userAttribute)
        
    # Добавляем дополнительный атрибут
    additionalAttribute = {
        "type": "additionalAttribute",
        "value": str(fpd_value),
        "print": True
    }
    output["items"].append(additionalAttribute)

    # Добавляем позиции
    for item in tlv.get("1059", []):
        position = {
            "type": "position",
            "name": item.get("1030", ""),  # Наименование
            "price": item.get("1079", 0) / 100,  # Цена
            "quantity": item.get("1023", 0),  # Количество
            "amount": item.get("1043", 0) / 100,  # Сумма
            "measurementUnit": "piece",  # По умолчанию штуки
            "paymentMethod": "fullPayment",  # По умолчанию полная оплата
            "paymentObject": "commodity",  # По умолчанию товар
            "tax": {"type": "none"}  # По умолчанию без НДС
        }
        output["items"].append(position)

    # Добавляем оплату
    cash_amount = tlv.get("1031", 0) / 100  # Сумма наличными
    electronic_amount = tlv.get("1081", 0) / 100  # Сумма безналичными
    
    total_amount = cash_amount + electronic_amount
    
    if cash_amount > 0:
        payment = {
            "type": "cash",
            "sum": cash_amount
        }
        output["payments"].append(payment)
        
    if electronic_amount > 0:
        payment = {
            "type": "electronically",
            "sum": electronic_amount
        }
        output["payments"].append(payment)
        
    if total_amount > 0:
        output["total"] = total_amount

    return output

def process_directory(input_dir, output_dir):
    """Обрабатывает все JSON файлы в указанной директории"""
    
    # Получаем абсолютные пути
    input_dir = os.path.abspath(input_dir)
    output_dir = os.path.abspath(output_dir)
    
    print(f"Обрабатываем директорию: {input_dir}")
    
    # Проверяем существование входной директории
    if not os.path.exists(input_dir):
        print(f"Ошибка: директория {input_dir} не существует")
        return
        
    # Создаем выходную директорию если её нет
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
        print(f"Создана директория: {output_dir}")
    
    # Получаем список всех JSON файлов
    json_files = glob.glob(os.path.join(input_dir, "*.json"))
    print(f"Найдено файлов: {len(json_files)}")
    
    if not json_files:
        print("Не найдено JSON файлов для обработки")
        return
        
    for json_file in json_files:
        print(f"Обрабатываем файл: {json_file}")
        try:
            # Читаем входной JSON
            with open(json_file, 'r', encoding='utf-8') as f:
                input_json = json.load(f)
            
            # Конвертируем JSON
            output_json = convert_json(input_json)
            
            if output_json:
                # Создаем имя выходного файла
                filename = os.path.basename(json_file)
                output_path = os.path.join(output_dir, filename)
                
                # Сохраняем преобразованный JSON
                with open(output_path, 'w', encoding='utf-8') as f:
                    json.dump(output_json, f, ensure_ascii=False, indent=2)
                    
                print(f"Успешно обработан файл: {filename}")
            else:
                print(f"Ошибка в файле {json_file}: неверный формат входного JSON")
                
        except Exception as e:
            print(f"Ошибка при обработке файла {json_file}: {str(e)}")

if __name__ == "__main__":
    # Получаем текущую директорию скрипта
    script_dir = os.path.dirname(os.path.abspath(__file__))
    
    # Формируем пути относительно директории скрипта
    input_directory = os.path.join(script_dir, "input_json")
    output_directory = os.path.join(script_dir, "output_json")
    
    process_directory(input_directory, output_directory) 