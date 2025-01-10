#!/bin/bash

ZABBIX_API_URL="http://<zabbix_server_ip>/api_jsonrpc.php"  # Замените на URL вашего Zabbix API
ZABBIX_AUTH_TOKEN="YOUR_AUTH_TOKEN"   # Замените на ваш authentication token

# Функция для выполнения запросов к Zabbix API
zabbix_api_request() {
  local method="$1"
  local params="$2"

  local data=$(jq -n \
    --arg method "$method" \
    --arg params "$params" \
    --arg auth "$ZABBIX_AUTH_TOKEN" \
   '{
      "jsonrpc": "2.0",
      "method": $method,
      "params": ($params | fromjson),
      "auth": $auth,
      "id": 1
    }')

  curl -s -H "Content-Type: application/json" \
    -d "$data" \
    "$ZABBIX_API_URL" | jq -r '.result'
}

main() {
  # 1. Получение ID группы "solar"
  local group_id=$(zabbix_api_request "hostgroup.get" '{"output": "groupids", "filter": {"name": ["solar"]}}' | jq -r '.[0]')

    if [ -z "$group_id" ]; then
    echo "Группа 'solar' не найдена."
    return 1
  fi

  # 2. Получение ID хостов в группе "solar"
  local host_ids_json=$(zabbix_api_request "host.get" '{"output": "hostids", "groupids": ["'"$group_id"'"]}')

  if [[ "$host_ids_json" == "null" || -z "$host_ids_json" ]]
  then
      echo "Нет хостов в группе 'solar'."
      return 0
  fi

  local host_ids_array=($(echo "$host_ids_json" | jq -r '.[] | .[]'))

  #3. Получение ID maintenance для хостов
  local maintenance_ids_json=$(zabbix_api_request "maintenance.get" '{"output": "maintenanceids", "hostids": ['"$(IFS=","; echo "${host_ids_array[*]}" )"']}');

  if [[ "$maintenance_ids_json" == "null" || -z "$maintenance_ids_json" ]]
  then
      echo "Нет режимов обслуживания для хостов группы 'solar'."
      return 0
  fi

  local maintenance_ids_array=($(echo "$maintenance_ids_json" | jq -r '.[] | .[]'))

  # 4. Удаление режима обслуживания
   local delete_result=$(zabbix_api_request "maintenance.delete" "[\"$(IFS='","'; echo \"${maintenance_ids_array[*]}\")\"]")

   if [ -n "$delete_result" ]; then
       local deleted_count=$(echo "$delete_result" | jq 'length')
        echo "Успешно удалено $deleted_count режимов обслуживания."
    else
        echo "Не удалось удалить режимы обслуживания."
    fi
}


main

