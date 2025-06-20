#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import re
import json
import logging
import shutil
import pdfplumber
from datetime import datetime
from typing import List, Dict, Any

# ————— Настройка логирования —————
LOG_FILE = "parse_receipts_errors.log"
logging.basicConfig(
    filename=LOG_FILE,
    filemode="w",
    encoding="utf-8",      # теперь лог будет в UTF-8
    level=logging.ERROR,
    format="%(asctime)s %(levelname)s %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

# шаблон "qty x price total"
QTY_RE    = re.compile(r"([\d.,]+)\s*[xX*]\s*([\d.,]+)\s*([\d.,]+)")

def extract_items(lines: List[str], start: int, end: int) -> List[Dict[str,Any]]:
    items: List[Dict[str,Any]] = []
    i = start
    while i < end:
        # 1) пропускаем маркеры
        ItsMarker = ("[М+]" in lines[i]) or ("[М-]" in lines[i]) or ("[М]" in lines[i])
        logging.info(lines[i])
        #print("--------------")
        #print(lines[i])
        while i < end and ItsMarker:
            i += 1
        if i >= end:
            break

        # 2) ищем строку с qty x price total
        j = i
        while j < end and not QTY_RE.fullmatch(lines[j]):
            j += 1
        if j >= end:
            # больше нет позиций
            break

        # 3) всё от i до j — это описание +, возможно, unit
        desc_lines = lines[i:j]
        #print(desc_lines)
        logging.info(desc_lines)

        # если в последней строке описания есть ';', разделяем
        ItsMarker = ("[М+]" in desc_lines[-1]) or ("[М-]" in desc_lines[-1]) or ("[М]" in desc_lines[-1])
        lastIndexDesc = -1
        if ItsMarker:
            lastIndexDesc = -2
        last = desc_lines[lastIndexDesc]
        #print("last="+last)
        logging.info(last)
        if ";" in last:
            text, maybe_unit = last.split(";", 1)
            desc_lines[lastIndexDesc] = text.strip()
            unit = maybe_unit.strip()
        # иначе, если последняя строка состоит из одиночного слова/токена — вероятно, это unit
        elif len(desc_lines) >= 2 and re.fullmatch(r"[^\s\d]+" , last):
            #print("len(desc_lines) >= 2 and re.fullmatch")
            unit = last
            #print(desc_lines)
            desc_lines = desc_lines[:-1]

        unit = None
        #print(desc_lines)
        # 4) собираем окончательное описание
        if ItsMarker:
            desc = " ".join(desc_lines[:-1]).strip()
        else:
            desc = " ".join(desc_lines).strip()
        #print("desc="+desc)

        # 5) парсим qty, price, total
        #print(lines[j])
        qty, price, total = QTY_RE.match(lines[j]).groups()

        # 6) следующие три строки — НДС, признак расчёта, признак предмета
        vat, payment_method, item_type = None, None, None
        #print("j+1="+lines[j+1])
        #print("j+2="+lines[j+2])
        #print("j+3="+lines[j+3])
        #print("j+4="+lines[j+4])
        if j+1 < end:
            m = re.search(r"НДС\s+(\d+)%", lines[j+1])
            if m: vat = int(m.group(1))
        if j+2 < end:
            m = re.search(r"Признак способа расчета\s+(.+)", lines[j+2])
            if m: payment_method = m.group(1)
        if j+3 < end:
            m = re.search(r"Признак предмета расчета\s+(.+)", lines[j+3])
            if m: item_type = m.group(1)

        items.append({
            "description":    desc,
            "unit":           unit,
            "quantity":       float(qty.replace(",", ".")),
            "unit_price":     float(price.replace(",", ".")),
            "sum":            float(total.replace(",", ".")),
            "vat_percent":    vat,
            "payment_method": payment_method,
            "item_type":      item_type,
        })

        # 7) двигаем i за весь разобранный блок
        i = j + 4

    return items

def extract_receipt_from_text(text: str) -> Dict[str, Any]:
    """
    Парсинг текста чека в словарь с реквизитами и списком позиций.
    """
    lines = [ln.strip() for ln in text.splitlines() if ln.strip()]
    data: Dict[str, Any] = {}
    # HEADER
    header = "\n".join(lines[:10])
    #если это отчет об открытии или закрытии смены, то пропускаем его
    if "Отчёт о закрытии смены" in header or "Отчёт об открытии смены" in header:
        return None
    # Название продавца
    data['seller_name'] = lines[0]
    # Номер чека, дата, время
    m = re.search(r"ЧЕК №:\s*(\d+)\s+(\d{2}\.\d{2}\.\d{2})\s+(\d{2}:\d{2})", header)
    if m:
        data['receipt_number'] = m.group(1)
        # Приводим дату к ISO
        dd,mm,yy = m.group(2).split(".")
        data['datetime'] = datetime.strptime(f"20{yy}-{mm}-{dd} {m.group(3)}", "%Y-%m-%d %H:%M").isoformat()
    # Смена
    m = re.search(r"СМЕНА:\s*(\d+)", header)
    if m: data['shift'] = m.group(1)
    # Кассир
    m = re.search(r"КАССИР\s+(.+)", header)
    if m: data['cashier'] = m.group(1)

    # ИНН
    m = re.search(r"ИНН\s+(\d+)", text)
    if m: data['inn'] = m.group(1)

    # Определение типа чека по заголовку
    data['receipt_type'] = None
    for ln in lines:
        if ln == "ПРИХОД":
            data['receipt_type'] = 'sale'
            break
        elif ln == "ВОЗВРАТ ПРИХОДА":
            data['receipt_type'] = 'return'
            break

    # Адрес — берём все строки между ИНН и ПРИХОД
    addr_lines = []
    idx_inn = next(i for i,ln in enumerate(lines) if ln.startswith("ИНН"))
    for ln in lines[idx_inn+1:]:
        if ln in ("ПРИХОД", "ВОЗВРАТ ПРИХОДА"):
            break
        addr_lines.append(ln)
    data['address'] = " / ".join(addr_lines)

    start = next(i for i, ln in enumerate(lines)
                     if ln in ("ПРИХОД", "ВОЗВРАТ ПРИХОДА")) + 1
    end = next(i for i, ln in enumerate(lines)
                   if ln.startswith("ИТОГО:"))

    if start is not None and end is not None and start < end:
        items = extract_items(lines, start, end)
    else:
        items = []        
    data['items'] = items

    # TOTALS & FOOTER
    # Итого
    m_total = re.search(r"ИТОГО:\s*([\d.,]+)", text)
    if m_total: data['total'] = float(m_total.group(1).replace(",", "."))
    # Наличными
    m_cash = re.search(r"(?<!\w)НАЛИЧНЫМИ:\s*([\d.,]+)", text)
    if m_cash: data['paid_cash'] = float(m_cash.group(1).replace(",", "."))
    # Безналичными
    m_card = re.search(r"БЕЗНАЛИЧНЫМИ:\s*([\d.,]+)", text)
    if m_card: data['paid_card'] = float(m_card.group(1).replace(",", "."))
    # НДС по чеку
    m_NDS = re.search(r"НДС\s+(\d+)%:\s*([\d.,]+)", text)
    if m_NDS:
        data['total_vat_percent'] = int(m_NDS.group(1))
        data['total_vat_sum']     = float(m_NDS.group(2).replace(",", "."))
    # остальные поля
    patterns = {
        "sender_email": r"ЭЛ\.АДР\.ОТПРАВИТЕЛЯ:\s*(\S+)",
        "fns_site":     r"САЙТ ФНС:\s*(\S+)",
        "tax_system":   r"СНО:\s*(.+)",
        "kkt_number":   r"№ ККТ:\s*(\d+)",
        "fn_number":    r"№ ФН:\s*(\d+)",
        "fd_number":    r"№ ФД:\s*(\d+)",
        "fp":           r"ФП\s*(\d+)",
        "qr_url":       r"https?://\S+"
    }
    for key, pat in patterns.items():
        m = re.search(pat, text)
        if m: data[key] = m.group(1)

    return data

def validate_and_log(data: Dict[str, Any], file_path: str):
    issues = []

    # 1) Нет позиций
    if not data.get('items'):
        issues.append("no positions parsed")

    # 2) Отсутствуют обязательные поля
    for fld in ('receipt_number', 'datetime', 'total'):
        if fld not in data or data[fld] is None:
            issues.append(f"missing field '{fld}'")

    # 3) Сумма по позициям vs total
    if data.get('items') and 'total' in data:
        items_sum = sum(item.get('sum', 0) for item in data['items'])
        if abs(items_sum - data['total']) > 0.01:
            issues.append(f"items sum {items_sum:.2f} ≠ total {data['total']:.2f}")

    # 4) Сумма оплат vs total
    paid_cash = data.get('paid_cash', 0.0)
    paid_non = data.get('paid_card', 0.0)
    if 'total' in data:
        payments_sum = paid_cash + paid_non
        if abs(payments_sum - data['total']) > 0.01:
            issues.append(
                f"payments sum {payments_sum:.2f} ≠ total {data['total']:.2f}"
            )
    #5) описание товара - пустое
    if data.get('items') and 'total' in data:
        for item in data['items']: 
            if item.get('description') == "":
                issues.append(f"пустое описание товара")

    # Записываем все найденные проблемы в лог
    for msg in issues:
        logging.error(f"{file_path}: {msg}")
    return issues

def process_pdf(file_path: str) -> Dict[str, Any]:
    """
    Читает PDF, извлекает весь текст и парсит его.
    """
    #print(f"Processing {file_path}…")
    with pdfplumber.open(file_path) as pdf:
        full_text = "\n".join(page.extract_text() or "" for page in pdf.pages)
    data = extract_receipt_from_text(full_text)
    if data == None:
        return None, None
    issues = validate_and_log(data, file_path)
    return data, issues

def main(input_folder: str, output_folder: str):
    error_folder = ("./errors")
    os.makedirs(error_folder, exist_ok=True)    
    """
    Если input_folder — файл, обрабатываем один PDF.
    Если папка — обрабатываем все PDF в ней и сохраняем список.
    """
    # Создаем директорию для сконвертированных файлов, если её нет
    if not os.path.exists(output_folder):
        os.makedirs(output_folder)

    for filename in os.listdir(input_folder):
        if not filename.lower().endswith('.pdf'):
            continue
        pdf_path = os.path.join(input_folder, filename)
        print(f"Обработка {pdf_path}...")
        resultjson, issues = process_pdf(pdf_path)

        if resultjson == None:
            continue

        # если были ошибки — копируем PDF в errors/
        if issues:
            shutil.copy(pdf_path, os.path.join(error_folder, filename))

        # сохраняем в JSON
        json_name = os.path.splitext(filename)[0] + '.json'
        json_path = os.path.join(output_folder, json_name)
        with open(json_path, "w", encoding="utf-8") as jf:
            json.dump(resultjson, jf, ensure_ascii=False, indent=2)
        print(f"Done. Results saved to {json_path}")

if __name__ == "__main__":
    import argparse
    p = argparse.ArgumentParser(description="Парсинг чеков из PDF")
    #p.add_argument("input",  help="PDF-файл или папка с PDF")
    #p.add_argument("output", help="Имя выходного JSON")
    args = p.parse_args()
    #inputfiles = args.input
    #outputfiles = args.output
    #if not args.input:
    #p.print_help()
    inputfiles = './taxcomePDF'
    outputfiles = './taxcomjsons'
    main(inputfiles, outputfiles)