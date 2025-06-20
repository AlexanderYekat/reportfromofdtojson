import json
import os
import glob
from pathlib import Path

def validate_checks():
    checks_dir = Path('./json')
    start_dir = os.path.abspath(checks_dir)
    total_sum = 0
    errors = []

    # Проходим по всем json файлам в директории checks

    #for file_path in glob.iglob(start_dir + '**/*.json', recursive=True):
    for dirpath, dirnames, filenames in os.walk(checks_dir):
        for filename in filenames:
            if not filename.lower().endswith('.json'):
                continue
            #for file_path in checks_dir.glob('*.json', recursive=True):
            file_path = os.path.join(dirpath, filename)
            try:
                #print(file_path)
                with open(file_path, 'r', encoding='utf-8') as f:
                    check = json.load(f)

                #print(check)
                #print("-------------------")
                items_sum=0
                # Считаем сумму по позициям
                for item in check['items']:
                    #print(item)
                    if item['type']=='position':
                        items_sum+=item['amount']
                #items_sum = sum(item['amount'] for item in check['items'])
                #continue
            
                # Считаем сумму по платежам
                payments_sum = sum(payment['sum'] for payment in check['payments'])

                # Добавляем к общей сумме
                total_sum += payments_sum

                # Проверяем соответствие сумм
                if items_sum != payments_sum:
                    errors.append(
                        f"Ошибка в файле {file_path.name}:\n"
                        f"  Сумма по позициям: {items_sum}\n"
                            f"  Сумма по платежам: {payments_sum}\n"
                        f"  Разница: {items_sum - payments_sum}"
                    )

            except Exception as e:
                errors.append(f"Ошибка при обработке файла {file_path.name}: {str(e)}")

    # Выводим результаты
    print(f"\nОбщая сумма всех чеков: {total_sum:,} руб.".replace(',', ' '))
    
    if errors:
        print("\nНайдены ошибки:")
        for error in errors:
            print(f"\n{error}")
    else:
        print("\nОшибок не найдено. Все суммы совпадают.")

if __name__ == "__main__":
    validate_checks()