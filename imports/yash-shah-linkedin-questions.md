# Interview questions from Yash Shah’s public LinkedIn posts

Collected October 14, 2026. Questions are paraphrased from publicly indexed posts. The activity feed could not be fetched directly, so this is a partial collection, not all activity. Post age does not establish an interview date. Review metadata before publishing; these have not been added to the database.

## Deloitte — Java Full Stack Consultant, L1, 3–5 years

Source: https://www.linkedin.com/posts/yash-shah-java-fullstack-developer_javadeveloper-javafullstack-springboot-activity-7381331576829276161-4z_T

1. Explain your project and backend responsibilities.
2. How do reference comparison and equals() differ?
3. Which methods does Object provide?
4. Explain throwing an exception versus declaring exceptions with throws.
5. Compare fail-fast and fail-safe iteration.
6. Describe how HashMap stores and retrieves entries.
7. Why should equals() and hashCode() be overridden together?
8. Which Java 8 capabilities have you used in production?
9. Compare intermediate and terminal stream operations.
10. How does the modern date/time API improve on java.util.Date?
11. Explain Optional and its use cases.
12. How do map() and flatMap() differ?
13. What makes an interface functional, and why is that useful?
14. Find every most-frequent value using ordinary Java and streams: [1,2,3,2,3,4,1,4,1,2,4,3,4,2].

The source separately asks what throw/throws are and how they differ; those repeated prompts are combined here.

## EY — Java Full Stack Developer, L1, 4–6 years

Source: https://www.linkedin.com/posts/yash-shah-java-fullstack-developer_ey-java-full-stack-developer-l1-interview-activity-7430245277761945600-zpVV

1. How would you keep microservices resilient when dependencies fail?
2. Which architecture patterns have you applied to microservices?
3. Explain circuit breakers and your Resilience4j implementation.
4. Describe CQRS and when you would use it.
5. Discuss your experience with Kafka or RabbitMQ.
6. How have you enforced roles and permissions with Spring Security?
7. Compare bean selection using @Qualifier and @Primary.
8. How do you read configured properties into a business service?
9. Compare implementing Runnable with extending Thread.
10. For threads sharing resources, explain wait() versus run().
11. Find pairs totaling 6 in [2,4,3,3,5,7], using Java or streams; expected pairs are [2,4] and [3,3].
12. Write SQL for employees paid above their department’s average.
13. Compare primary-key and unique-key constraints.

The source says wait() versus run(); this potentially unusual wording is retained for review.

## Hitachi Digital Services — Java Full Stack Developer, L1 Screening

Source: User-provided LinkedIn post text from Yash Shah. The post was marked “7mo • Edited”; no direct post URL or interview date was provided.

1. Describe yourself and your recent project architecture, tech stack and role.
2. Which Java version and features have you used in real projects?
3. Find the second-highest unique integer in an array using Java 8 Streams; explain the pipeline, sorting, distinct elements and terminal operation.
4. Explain HashMap hashing, collision handling and its linked-list/tree bucket structure.
5. What is multiple inheritance in Java, and how do interfaces support it?
6. Explain the Singleton pattern and show how a Java implementation ensures a single instance.
7. How do you implement RBAC in Spring Boot? How does Spring Security validate roles and restrict API access?
8. What is the Spring IoC Container, and how does dependency injection work internally?
9. What is Angular lazy loading, and how does it improve performance?
10. How do you optimize Angular page load performance using lazy loading, change detection, TrackBy and code splitting?

## Wipro — Java Full Stack Developer, 3–5 years

Source: User-provided LinkedIn post text from Yash Shah. The post was marked “7mo”; no direct post URL or interview date was provided. Duration: 45–60 minutes.

1. Describe yourself and your roles and responsibilities in your project.
2. How many years of experience do you have with Java, Angular, MySQL, Kafka and AWS?
3. Explain the SOLID principles with real-world examples.
4. Compare map() and flatMap() in Java Streams, including efficiency and use cases.
5. Explain ConcurrentHashMap internals and its multithreaded performance benefits.
6. Explain Java memory management and memory types.
7. Compare heap and stack memory and their performance impact.
8. What is the transient keyword, and why is it used?
9. Explain synchronization in Java with an example.
10. What is ClassNotFoundException, and when does it occur?
11. Explain how @EnableAutoConfiguration works internally in Spring Boot.
12. What is autowiring, and what are alternative dependency injection methods?
13. What steps connect a Spring Boot application to a database?
14. Write a REST API to handle an ID payload and implement CRUD mapping.
15. Explain the Saga pattern in microservices with an example.
16. What is CQRS (Command Query Responsibility Segregation)?
17. Compare the SQL WHERE and HAVING clauses.
18. What is a self-join in SQL?
19. Explain inner and outer queries in a subquery.
20. Compare LEFT JOIN and RIGHT JOIN.
21. How do @Component and @Primary differ in Spring Boot?
22. How do you implement an Angular AuthGuard to secure routes?

## Paramatrix Technologies Pvt Ltd — Java Full Stack Developer, L1, 3–5 years

Source: User-provided LinkedIn post text from Yash Shah. The post was marked “8mo • Edited”; no direct post URL or interview date was provided. Duration: 20–30 minutes.

1. Discuss recent project architecture, technology stack, responsibilities, production deployment experience, and handling performance issues or production bugs.
2. Find the second-highest number in [2,3,4,5,6,1,2,3,6,7,8,9,8], using both traditional Java without Java 8 and Java 8 Streams.
3. Given a parent method with protected visibility and an overriding child method with default visibility, determine whether it compiles. Explain visibility and return-type rules for overriding.
4. Discuss Java versions used, Java 8 features, improvements in Java 17/21, and experience migrating projects.
5. Identify Spring Boot versions compatible with Java 17 and 21, and explain upgrade steps covering build configuration, dependencies, compiler settings and deprecated APIs.
6. Discuss Angular versions used, whether Angular depends on the Java backend version, and how to ensure frontend/backend compatibility.
7. Find the eighth-highest salary using a subquery and DENSE_RANK() or ROW_NUMBER(), accounting for duplicate salaries.

## Hexaware Technologies — AI TalentQuest Java Full Stack Developer, 3–5 years

Source: User-provided LinkedIn post text from Yash Shah. The post was marked “8mo”; no direct post URL or interview date was provided. Duration: one hour. Screening focused on JWT security, microservices, Angular and logging.

1. Explain project architecture and stack, including Angular, Spring Boot, microservices and MySQL.
2. How do you implement JWT authentication to secure Spring Boot REST APIs?
3. Explain JWT generation, validation, filters and Spring Security integration.
4. How do you implement JWT-based RBAC for ADMIN and USER roles?
5. Design a secure authentication system using Spring Security and JWT.
6. What is Java logging, and why does it matter in production?
7. Compare Log4j2 and SLF4J.
8. Configure log4j2.xml to write error logs to a file and debug logs to the console.
9. How do microservices communicate using Feign Client?
10. How do you configure and enable Feign Client in Spring Boot?
11. Compare Feign Client and RestTemplate.
12. How do you implement lazy loading in Angular modules?
13. How do you manage shared Angular services and avoid duplication?
14. Build an Angular component that displays a product list, filters by search, and uses Reactive Forms.

## Deloitte — Java Full Stack Developer, L1, 1–3 years

Source: https://www.linkedin.com/posts/yash-shah-java-fullstack-developer_deloitte-java-javadeveloper-activity-7290696692440469504-KA5S

1. Describe your experience, projects and contributions.
2. Compare HashMap with Hashtable, including concurrency and null handling.
3. Compare ArrayList and LinkedList storage and modification costs.
4. Explain Angular’s two-way data binding.
5. What monitoring capabilities does Spring Boot Actuator provide?
6. Implement numeric reversal in Java.

## Deloitte — Java Developer, L1, 3–5 years

Source: https://www.linkedin.com/posts/yash-shah-java-fullstack-developer_deloitte-big4-javafullstack-activity-7283686569058652160-GYsc

1. Introduce yourself and describe your projects.
2. Explain your responsibilities and the functionality you delivered.
3. For employees belonging to departments, where should the foreign key reside?
4. Return each department’s name and employee count using SQL.
5. Model employees belonging to multiple departments; explain keys, storage and the query.
6. How does an employee without a department affect the previous query?
7. How would null employee or department values affect the result count?
