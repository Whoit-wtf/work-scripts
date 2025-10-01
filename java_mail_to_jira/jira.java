import com.atlassian.jira.rest.client.api.JiraRestClient;
import com.atlassian.jira.rest.client.api.AsynchronousJiraRestClientFactory;
import java.net.URI;

public class JiraService {
    private static final String JIRA_URL = "http://your-jira-instance.com";
    private static final String USERNAME = "your_username";
    private static final String PASSWORD = "your_password_or_api_token"; // Рекомендуется использовать API-токен

    public JiraRestClient login() {
        AsynchronousJiraRestClientFactory factory = new AsynchronousJiraRestClientFactory();
        URI jiraServerUri = URI.create(JIRA_URL);
        return factory.createWithBasicHttpAuthentication(jiraServerUri, USERNAME, PASSWORD);
    }
}

import com.atlassian.jira.rest.client.api.JiraRestClient;
import com.atlassian.jira.rest.client.api.domain.BasicIssue;
import com.atlassian.jira.rest.client.api.domain.input.IssueInput;
import com.atlassian.jira.rest.client.api.domain.input.IssueInputBuilder;
import java.util.concurrent.ExecutionException;

public BasicIssue createSimpleIssue() throws ExecutionException, InterruptedException {
    try (JiraRestClient restClient = login()) {
        
        IssueInputBuilder issueBuilder = new IssueInputBuilder();
        
        // Обязательные поля
        issueBuilder.setProjectKey("TEST") // Ключ проекта
                   .setIssueTypeId(3L)     // ID типа задачи (например, 3 для "Задачи")
                   .setSummary("Краткое описание задачи");
        
        // Дополнительные поля
        issueBuilder.setDescription("Детальное описание задачи");
        
        IssueInput issueInput = issueBuilder.build();
        
        // Создание задачи
        BasicIssue createdIssue = restClient.getIssueClient()
                                           .createIssue(issueInput)
                                           .get();
        
        System.out.println("Задача создана: " + createdIssue.getKey());
        return createdIssue;
    }
}