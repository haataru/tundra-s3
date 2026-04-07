# tundra-s3

HTTP-сервер с S3-совместимым API для хранения файлов на диске.

## Быстрый старт

```bash
# Запуск сервера
./src/s3-server.exe -port=8080

# Загрузить файл
curl -X PUT -T audio.mp3 http://localhost:8080/mybucket/audio.mp3

# Скачать файл
curl http://localhost:8080/mybucket/audio.mp3 -o audio.mp3

# Список файлов
curl http://localhost:8080/mybucket/

# Удалить бакет
curl -X DELETE http://localhost:8080/mybucket/
```

## Параметры запуска

Параметр `port`: значение по умолчанию `8080`, это порт сервера.

Параметр `data-dir`: значение по умолчанию `./data`, это директория для хранения файлов.

Параметр `max-concurr`: значение по умолчанию `500`, это максимальное количество одновременных запросов.

## API

### PUT /{bucket}/{key} - Загрузить файл

```bash
curl -X PUT -T file.txt http://localhost:8080/mybucket/file.txt
curl -X PUT --data-binary @image.jpg http://localhost:8080/mybucket/image.jpg
```

### GET /{bucket}/{key} - Скачать файл

```bash
# В консоль
curl http://localhost:8080/mybucket/file.txt

# В файл
curl http://localhost:8080/mybucket/file.txt -o file.txt

# Частичная загрузка (Range)
curl -H "Range: bytes=0-1023" http://localhost:8080/mybucket/file.txt
```

### HEAD /{bucket}/{key} - Метаданные

```bash
curl -I http://localhost:8080/mybucket/file.txt
```

### DELETE /{bucket}/{key} - Удалить объект

```bash
curl -X DELETE http://localhost:8080/mybucket/file.txt
# Ответ: 204 No Content
```

### DELETE /{bucket}/ - Удалить бакет

```bash
curl -X DELETE http://localhost:8080/mybucket/
# Ответ: 204 No Content
```

### GET /{bucket}/ - Список объектов

```bash
# Все файлы
curl http://localhost:8080/mybucket/

# С префиксом
curl "http://localhost:8080/mybucket/?prefix=docs/"

# С delimiter (эмуляция папок)
curl "http://localhost:8080/mybucket/?delimiter=/"
```

## Валидация

### Имя бакета
- Латиница a-z, цифры 0-9, дефис
- 3-63 символа
- Начинается и заканчивается буквой или цифрой

### Ключ
- Максимум 1024 символа
- Не может начинаться с /
- Запрещены ".." (защита от path traversal)

## Структура

```
src/
├── main.go         # Точка входа, парсинг аргументов
├── server/         # HTTP слой
│   ├── server.go   # Server с семафором (max 500 горутин)
│   ├── handlers.go # Обработчики PUT/GET/DELETE/HEAD/LIST
│   └── routing.go  # Маршрутизация
├── storage/        # Слой хранения
│   ├── storage.go  # DiskStorage (Put/Get/Delete/List)
│   └── stream.go   # ETag (MD5)
├── types/          # Интерфейсы и типы
│   └── types.go    # Storage interface, ObjectInfo
└── utils/          # Утилиты
    ├── validate.go # Валидация бакета и ключа
    └── path.go     # Безопасные пути
```

## Хранение

```
data/                    # создается автоматически
├── .tmp/                # временные файлы при загрузке
└── {bucket}/
    └── {key}            # data/mybucket/file.txt
```

## Ограничения прототипа

- Нет аутентификации/авторизации
- Нет версионирования
- Нет репликации
- Нет multipart upload
- Нет сжатия/шифрования

## Требования

- Go 1.25+
- Windows/Linux/macOS
