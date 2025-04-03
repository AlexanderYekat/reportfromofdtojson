import json
import os
import argparse
from datetime import datetime

def convert_tax_type(tax_type):
    """Конвертирует тип налогообложения"""
    tax_mapping = {
        32: "patent",
        #0: "osn",
        #1: "usnIncome",
        2: "usnIncome",
        #4: "envd",
        #16: "esn"
        # Добавьте другие соответствия по необходимости
    }
    return tax_mapping.get(tax_type)

def convert_payment_method(calculation_method):
    """Конвертирует метод оплаты"""
    method_mapping = {
        4: "fullPayment",
        # Добавьте другие соответствия по необходимости
    }
    return method_mapping.get(calculation_method, "fullPayment")

def convertNDSRate(ndsRate):
    """Конвертирует ставку НДС"""
    ndsRate_mapping = {
        6: "none",
        1: "vat20",
        2: "vat10",
        7: "vat5",
        8: "vat7",
        5: "vat0",
    }
    return ndsRate_mapping.get(ndsRate)

def convert_items(items):
    """Конвертирует позиции чека"""
    converted_items = []
    
    for item in items:
        if item["Price"] == 0:  # Пропускаем позиции с нулевой ценой
            continue

        ndsRate = convertNDSRate(item["NDS_Rate"])

        converted_item = {
            "type": "position",
            "name": item["Name"],
            "price": item["Price"] / 100,  # Конвертируем копейки в рубли
            "quantity": item["Quantity"],
            "amount": item["Total"] / 100,  # Конвертируем копейки в рубли
            "measurementUnit": "piece",
            "paymentMethod": convert_payment_method(item["CalculationMethod"]),
            "paymentObject": "commodity",
            "tax": {
                "type": ndsRate
            }
        }
        converted_items.append(converted_item)
    
    return converted_items

def convert_type(operationType):
    type_mapping = {
        1: "sellCorrection",
        2: "sellReturnCorrection",
        3: "buyCorrection",
        4: "buyReturnCorrection"
    }
    return type_mapping.get(operationType)

def stornoTypeCheck(typecheck):
    if typecheck == "sellCorrection":
        return "sellReturnCorrection"
    elif typecheck == "sellReturnCorrection":
        return "sellCorrection"
    elif typecheck == "buyCorrection":
        return "buyReturnCorrection"
    elif typecheck == "buyReturnCorrection":
        return "buyCorrection"
    else:
        return typecheck

def change_tax_type():
    return "patent "

def convert_fetch_to_atol(input_json, storno=False, changeOSN=False):
    """Конвертирует JSON из формата fetch в формат atol"""
    doc = input_json["Document"]
    
    total_amount = doc["Amount_Total"] / 100  # Конвертируем копейки в рубли
    
    # Конвертируем обычные позиции
    regular_items = convert_items(doc["Items"])
    typecheck = convert_type(doc["OperationType"])

    if storno:
        typecheck = stornoTypeCheck(typecheck)

    taxationType = convert_tax_type(doc["TaxationType"])

    if changeOSN:
        taxationType = change_tax_type()
    
    # Добавляем дополнительные атрибуты с реальными значениями из документа
    additional_items = [
        {
            "type": "userAttribute",
            "name": "ФД",
            "value": input_json["DocNumber"],  # Номер ФД из документа
            "print": True
        },
        {
            "type": "additionalAttribute",
            "value": str(doc["DecimalFiscalSign"]),  # Фискальный признак из документа
            "print": True
        }
    ]
    
    result = {
        "type": typecheck,
        "electronically": True,
        "taxationType": taxationType,
        "correctionType": "self",
        "correctionBaseDate": doc['DateTime'][:10].replace('-', '.'),
        "operator": {
            "name": doc["Operator"]
        },
        "items": additional_items + regular_items,
        "payments": [
            {
                "type": "electronically",
                "sum": total_amount
            }
        ],
        "total": total_amount
    }
    
    return result

def process_directory(storno=False, changeOSN=False):
    """Обрабатывает все JSON файлы в директории receipts"""
    receipts_dir = "./receipts"
    output_dir = "./converted_receipts"
    
    # Создаем директорию для сконвертированных файлов, если её нет
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
    
    # Перебираем все файлы в директории
    for filename in os.listdir(receipts_dir):
        if filename.endswith(".json"):
            input_path = os.path.join(receipts_dir, filename)
            output_path = os.path.join(output_dir, f"{filename}")
            
            try:
                # Читаем входной файл
                with open(input_path, 'r', encoding='utf-8') as f:
                    input_data = json.load(f)
                
                # Конвертируем данные
                converted_data = convert_fetch_to_atol(input_data, storno, changeOSN)
                
                # Сохраняем результат
                with open(output_path, 'w', encoding='utf-8') as f:
                    json.dump(converted_data, f, ensure_ascii=False, indent=2)
                    
                print(f"Успешно сконвертирован файл: {filename}")
                
            except Exception as e:
                print(f"Ошибка при обработке файла {filename}: {str(e)}")

if __name__ == "__main__":
    # Создаем парсер аргументов командной строки
    parser = argparse.ArgumentParser(description='Конвертер чеков из формата fetch в формат atol')
    parser.add_argument('--storno', action='store_true', help='Флаг сторно')
    parser.add_argument('--changeOSN', action='store_true', help='Флаг изменения ОСН')
    
    # Парсим аргументы
    args = parser.parse_args()
    
    # Запускаем обработку директории
    process_directory(args.storno, args.changeOSN)
    
    # Флаг сторно доступен как args.storno
    # Пример использования:
    if args.storno:
        print("Режим сторно активирован")

    if args.changeOSN:
        print("Режим изменения ОСН активирован")