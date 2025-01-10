#!/bin/bash

# Zabbix API settings
ZABBIX_URL="http://your.zabbix.server/api_jsonrpc.php" # Замените на свой URL
ZABBIX_USER="your_zabbix_user" # Замените на своего пользователя
ZABBIX_PASS="your_zabbix_password" # Замените на свой пароль

# Проверка наличия имени режима обслуживания в параметрах
if [ -z "$1" ]; then
  echo "Использование: $0 <имя режима обслуживания>"
  exit 1
fi

MAINTENANCE_NAME="$1"

# Функция для получения maintenance ID по имени
get_maintenance_id() {
    local name="$1"
    local json_data=$(cat << EOF
{
  "jsonrpc": "2.0",
  "method": "maintenance.get",
  "params": {
      "output": ["maintenanceid"],
       "filter": {
        "name": "$name"
       }
  },
  "auth": null,
  "id": 1
}
EOF
)

    local response=$(curl -s -H "Content-Type: application/json" -u "${ZABBIX_USER}:${ZABBIX_PASS}" -d "$json_data" "$ZABBIX_URL")
     local status_code=$(echo "$response" | jq -r '.result | length')
    if [ "$status_code" -eq 0 ]; then
      echo ""
    else
      echo "$response" | jq -r '.result[0].maintenanceid'
    fi
}

# Функция для удаления режима обслуживания
delete_maintenance() {
    local id="$1"
    local json_data=$(cat << EOF
{
  "jsonrpc": "2.0",
  "method": "maintenance.delete",
  "params": {
        "maintenanceids": ["$id"]
  },
  "auth": null,
  "id": 1
}
EOF
)

    local response=$(curl -s -H "Content-Type: application/json" -u "${ZABBIX_USER}:${ZABBIX_PASS}" -d "$json_data" "$ZABBIX_URL")
    local status_code=$(echo "$response" | jq -r '.result.maintenanceids[0]')
    if [ -z "$status_code" ]; then
      echo "Error delete maintenance ID $id: $response" >&2
      return 1
    fi
     echo "Maintenance with ID $id deleted"
}

# Получение ID режима обслуживания
MAINTENANCE_ID=$(get_maintenance_id "$MAINTENANCE_NAME")

if [ -z "$MAINTENANCE_ID" ]; then
    echo "Режим обслуживания с именем '$MAINTENANCE_NAME' не найден."
    exit 1
fi

# Удаление режима обслуживания
delete_maintenance "$MAINTENANCE_ID"

if [ $? -eq 0 ]; then
  echo "Режим обслуживания с именем '$MAINTENANCE_NAME' успешно удален."
else
    echo "Ошибка при удалении режима обслуживания '$MAINTENANCE_NAME'." >&2
    exit 1
fi

exit 0
