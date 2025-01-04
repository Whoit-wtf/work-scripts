#!/bin/bash

# Zabbix API параметры
ZABBIX_URL="http://your.zabbix.server/api_jsonrpc.php"
ZABBIX_USER="your_zabbix_user"
ZABBIX_PASS="your_zabbix_password"

# Проверка наличия параметров
if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Использование: $0 <группа хостов> <включить/выключить>"
  exit 1
fi

HOST_GROUP="$1"
ACTION="$2"

# Проверка корректности действия
if [[ "$ACTION" != "включить" && "$ACTION" != "выключить" ]]; then
  echo "Некорректное действие: $ACTION. Допустимые значения: включить, выключить"
  exit 1
fi

# Получение ID группы хостов
GROUP_ID=$(curl -s -H "Content-Type: application/json" -d "{\"jsonrpc\":\"2.0\",\"method\":\"hostgroup.get\",\"params\":{\"filter\":{\"name\":\"$HOST_GROUP\"}},\"id\":1}" "$ZABBIX_URL" | jq -r '.result[0].groupid')

if [ -z "$GROUP_ID" ]; then
  echo "Группа хостов '$HOST_GROUP' не найдена."
  exit 1
fi

# Создание JSON запроса
if [[ "$ACTION" == "включить" ]]; then
  STATUS=1
  MESSAGE="Включен режим обслуживания"
else
  STATUS=0
  MESSAGE="Выключен режим обслуживания"
fi

JSON_DATA=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "method": "maintenance.create",
  "params": {
    "name": "Автоматическое обслуживание от скрипта",
    "active_since": "${EPOCH_TIME}",
    "active_till": "0",
    "timeperiods": [
      {
        "hostids": [
          "$GROUP_ID"
        ],
        "days": [
          1,2,3,4,5,6,7
        ],
        "time": "00:00-24:00"
      }
    ],
    "status": $STATUS,
    "description": "$MESSAGE"
  },
  "id": 1
}
EOF
)

EPOCH_TIME=$(date +%s)

# Отправка запроса к Zabbix API
RESPONSE=$(curl -s -H "Content-Type: application/json" -u "$ZABBIX_USER:$ZABBIX_PASS" -d "$JSON_DATA" "$ZABBIX_URL")

# Проверка ответа
STATUS_CODE=$(echo "$RESPONSE" | jq -r '.result.maintenanceids')

if [[ -n "$STATUS_CODE" ]]; then
  echo "Режим обслуживания $ACTION для группы '$HOST_GROUP' успешно."
  # Необязательно, вывод ID режима обслуживания
  echo "ID режима обслуживания: $STATUS_CODE"
else
  echo "Ошибка при $ACTION режима обслуживания для группы '$HOST_GROUP':"
  echo "$RESPONSE"
  exit 1
fi

exit 0