1. создать пользователя app `adduser app`
2. добавить ключ `ssh-copy-id -i ~/.ssh/id_ed25519.pub app@host`
3. настроить sshd `/etc/ssh/sshd_config & /etc/ssh/sshd_config.d/*`
   ```bash
   PasswordAuthentication no 
   PermitRootLogin no
   ```
4. правим hosts `echo "127.0.0.1 $(hostname)" | sudo tee -a /etc/hosts > /dev/nul`
5. создать папку для приложения `sudo mkdir /app/`
6. поменять владельша и права доступа к ней `sudo chown app:app /app/ && sudo chmod 750 /app/`
7. устанавливаем ngrok `https://dashboard.ngrok.com/get-started/setup/linux`
8. делаем ngrok сервисом с конфигом `https://dashboard.ngrok.com/cloud-edge/edges`
   * Edge > Routes > Overview > Start a Tunnel > Start a tunnel from a config file
   * копируем конфиг в `/home/app/.config/ngrok/ngrok.yml`
9. ставим PG
   * `sudo apt install postgresql`
   * `sudo -i -u postgres psql`
   * `CREATE ROLE app WITH LOGIN PASSWORD 'app';`
   * `ALTER ROLE app WITH SUPERUSER;`
   * `CREATE DATABASE app WITH OWNER app;`
10. копируем релиз на сервер `curl -L -o runout https://github.com/Hrom-in-space/runOut/releases/download/v2/runout-linux-amd64`
11. `chmod +x runout`
12. подготовить env файл `touch /app/env` и заполнить его
13. системд настройка
- `runout.service` в `/etc/systemd/system/
- `sudo systemctl daemon-reload`
- `sudo systemctl enable runout.service`
- `sudo systemctl start runout.service`
- `sudo systemctl stop runout.service`
- `sudo systemctl status runout.service`
14. сделать команды без sudo `sudo visudo` и добавить `app ALL=(ALL) NOPASSWD: /bin/systemctl start runout.service, /bin/systemctl stop runout.service`
15. настроить доступ к серверу по ключу для деплоя
ssh-keygen -f ./app-deploy-key -t ed25519
ssh-copy-id -i ./app-deploy-key app@host
16. установить тулу миграции `https://github.com/golang-migrate/migrate`
