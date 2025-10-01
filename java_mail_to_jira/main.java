import java.util.Properties;
import javax.mail.*;
import javax.mail.internet.InternetAddress;

public class POP3EmailReader {

    public static void main(String[] args) {
        // Данные для подключения
        String host = "pop.gmail.com"; // Например, pop.gmail.com
        String port = "995";
        String userName = "your_email@example.com";
        String password = "your_password";
        
        // Вызов метода для получения писем
        receiveEmails(host, port, userName, password);
    }

    public static void receiveEmails(String host, String port, 
                                   String userName, String password) {
        Properties properties = new Properties();
        
        // Настройка свойств для POP3S (POP3 с SSL) :cite[1]:cite[4]
        properties.put("mail.pop3.host", host);
        properties.put("mail.pop3.port", port);
        properties.put("mail.pop3.starttls.enable", "true");
        properties.put("mail.pop3.ssl.enable", "true");
        properties.put("mail.pop3.socketFactory.class", "javax.net.ssl.SSLSocketFactory");
        properties.put("mail.pop3.socketFactory.fallback", "false");
        
        // Создание сессии :cite[6]
        Session session = Session.getDefaultInstance(properties);
        
        Store store = null;
        Folder folder = null;
        
        try {
            // Получение хранилища POP3S и подключение :cite[6]
            store = session.getStore("pop3s");
            store.connect(host, userName, password);
            
            // Открытие папки INBOX :cite[6]
            folder = store.getFolder("INBOX");
            folder.open(Folder.READ_ONLY);
            
            // Получение списка сообщений :cite[6]
            Message[] messages = folder.getMessages();
            System.out.println("Найдено писем: " + messages.length);
            
            // Обработка каждого сообщения
            for (int i = 0; i < messages.length; i++) {
                Message msg = messages[i];
                System.out.println("\n--- Сообщение #" + (i + 1) + " ---");
                
                // Отправитель
                Address[] fromAddresses = msg.getFrom();
                String from = (fromAddresses.length > 0) ? 
                    ((InternetAddress) fromAddresses[0]).getAddress() : "неизвестно";
                System.out.println("От: " + from);
                
                // Тема
                String subject = msg.getSubject();
                System.out.println("Тема: " + subject);
                
                // Дата отправки
                String sentDate = msg.getSentDate().toString();
                System.out.println("Дата: " + sentDate);
                
                // Содержимое письма :cite[6]
                String contentType = msg.getContentType();
                if (contentType.contains("text/plain") || contentType.contains("text/html")) {
                    try {
                        Object content = msg.getContent();
                        if (content != null) {
                            System.out.println("Содержимое: " + content.toString());
                        }
                    } catch (Exception ex) {
                        System.out.println("Ошибка при чтении содержимого: " + ex.getMessage());
                    }
                }
                System.out.println("----------------------------");
            }
            
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            // Закрытие ресурсов :cite[6]
            try {
                if (folder != null && folder.isOpen()) {
                    folder.close(false);
                }
                if (store != null && store.isConnected()) {
                    store.close();
                }
            } catch (MessagingException e) {
                e.printStackTrace();
            }
        }
    }
}