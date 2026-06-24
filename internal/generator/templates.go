package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func has(deps []string, id string) bool {
	for _, d := range deps {
		if d == id {
			return true
		}
	}
	return false
}

const (
	postgresBlock = `
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: %s
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
`
	mysqlBlock = `
  mysql:
    image: mysql:8
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: %s
      MYSQL_USER: user
      MYSQL_PASSWORD: password
    ports:
      - "3306:3306"
    volumes:
      - mysqldata:/var/lib/mysql
`
	redisBlock = `
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
`
	neo4jBlock = `
  neo4j:
    image: neo4j:5
    environment:
      NEO4J_AUTH: neo4j/password
    ports:
      - "7474:7474"
      - "7687:7687"
`
	rabbitBlock = `
  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"
`
	kafkaBlock = `
  kafka:
    image: confluentinc/cp-kafka:latest
    ports:
      - "9092:9092"
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_LISTENERS: PLAINTEXT://0.0.0.0:29092,PLAINTEXT_HOST://0.0.0.0:9092
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    depends_on:
      - zookeeper
  zookeeper:
    image: confluentinc/cp-zookeeper:latest
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000
    ports:
      - "2181:2181"
`
)

// InjectDocker writes a Dockerfile (multi-stage) and a docker-compose.yml that
// wires an `app` service to the backing services it spins up.
func InjectDocker(dest string, cfg Config) error {
	deps := cfg.Deps
	java := cfg.JavaVersion
	if java == "" {
		java = "21"
	}
	db := FolderName(cfg)

	var svc strings.Builder
	var depends, appEnv, volumes []string

	// Datasource: postgres unless mysql/h2 chosen. h2 is embedded (no service).
	switch {
	case has(deps, "postgresql") || (has(deps, "data-jpa") && !has(deps, "mysql") && !has(deps, "h2")):
		svc.WriteString(fmt.Sprintf(postgresBlock, db))
		depends = append(depends, "postgres")
		volumes = append(volumes, "pgdata")
		appEnv = append(appEnv,
			fmt.Sprintf("      SPRING_DATASOURCE_URL: jdbc:postgresql://postgres:5432/%s", db),
			"      SPRING_DATASOURCE_USERNAME: user",
			"      SPRING_DATASOURCE_PASSWORD: password")
	case has(deps, "mysql"):
		svc.WriteString(fmt.Sprintf(mysqlBlock, db))
		depends = append(depends, "mysql")
		volumes = append(volumes, "mysqldata")
		appEnv = append(appEnv,
			fmt.Sprintf("      SPRING_DATASOURCE_URL: jdbc:mysql://mysql:3306/%s", db),
			"      SPRING_DATASOURCE_USERNAME: user",
			"      SPRING_DATASOURCE_PASSWORD: password")
	}

	if has(deps, "data-redis") {
		svc.WriteString(redisBlock)
		depends = append(depends, "redis")
		appEnv = append(appEnv, "      SPRING_DATA_REDIS_HOST: redis")
	}
	if has(deps, "data-neo4j") {
		svc.WriteString(neo4jBlock)
		depends = append(depends, "neo4j")
		appEnv = append(appEnv,
			"      SPRING_NEO4J_URI: bolt://neo4j:7687",
			"      SPRING_NEO4J_AUTHENTICATION_USERNAME: neo4j",
			"      SPRING_NEO4J_AUTHENTICATION_PASSWORD: password")
	}
	if has(deps, "amqp") {
		svc.WriteString(rabbitBlock)
		depends = append(depends, "rabbitmq")
		appEnv = append(appEnv, "      SPRING_RABBITMQ_HOST: rabbitmq")
	}
	if has(deps, "kafka") {
		svc.WriteString(kafkaBlock)
		depends = append(depends, "kafka")
		// In-network clients use the internal listener (29092); 9092 is host-only.
		appEnv = append(appEnv, "      SPRING_KAFKA_BOOTSTRAP_SERVERS: kafka:29092")
	}

	// Assemble compose: app service first, then backing services, then volumes.
	var compose strings.Builder
	compose.WriteString("services:\n")
	compose.WriteString("  app:\n    build: .\n    ports:\n      - \"8080:8080\"\n")
	if len(depends) > 0 {
		compose.WriteString("    depends_on:\n")
		for _, d := range depends {
			compose.WriteString("      - " + d + "\n")
		}
	}
	if len(appEnv) > 0 {
		compose.WriteString("    environment:\n")
		compose.WriteString(strings.Join(appEnv, "\n") + "\n")
	}
	compose.WriteString(svc.String())
	if len(volumes) > 0 {
		compose.WriteString("\nvolumes:\n")
		for _, v := range volumes {
			compose.WriteString("  " + v + ":\n")
		}
	}

	if err := os.WriteFile(filepath.Join(dest, "docker-compose.yml"), []byte(compose.String()), 0644); err != nil {
		return err
	}

	// Multi-stage Dockerfile using the project's build wrapper.
	var dockerfile string
	if strings.HasPrefix(cfg.BuildTool, "gradle") {
		dockerfile = fmt.Sprintf(`FROM eclipse-temurin:%s-jdk AS build
WORKDIR /app
COPY . .
RUN chmod +x gradlew && ./gradlew --no-daemon -x test clean bootJar

FROM eclipse-temurin:%s-jre
WORKDIR /app
COPY --from=build /app/build/libs/*.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java","-jar","app.jar"]
`, java, java)
	} else {
		dockerfile = fmt.Sprintf(`FROM eclipse-temurin:%s-jdk AS build
WORKDIR /app
COPY . .
RUN chmod +x mvnw && ./mvnw -q -DskipTests clean package

FROM eclipse-temurin:%s-jre
WORKDIR /app
COPY --from=build /app/target/*.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java","-jar","app.jar"]
`, java, java)
	}

	return os.WriteFile(filepath.Join(dest, "Dockerfile"), []byte(dockerfile), 0644)
}

// InjectProperties appends runtime configuration (datasource, JPA, redis,
// actuator) to application.properties so the generated app actually starts.
func InjectProperties(dest string, cfg Config) error {
	deps := cfg.Deps
	db := FolderName(cfg)

	var b strings.Builder
	b.WriteString("\n# --- Added by Sprout ---\n")

	if has(deps, "data-jpa") {
		switch {
		case has(deps, "mysql"):
			b.WriteString(fmt.Sprintf("spring.datasource.url=jdbc:mysql://localhost:3306/%s\n", db))
			b.WriteString("spring.datasource.username=user\nspring.datasource.password=password\n")
		case has(deps, "h2"):
			b.WriteString("spring.datasource.url=jdbc:h2:mem:" + db + "\n")
			b.WriteString("spring.h2.console.enabled=true\n")
		default: // postgres
			b.WriteString(fmt.Sprintf("spring.datasource.url=jdbc:postgresql://localhost:5432/%s\n", db))
			b.WriteString("spring.datasource.username=user\nspring.datasource.password=password\n")
		}
		b.WriteString("spring.jpa.hibernate.ddl-auto=update\nspring.jpa.show-sql=true\n")
	}

	if has(deps, "data-redis") {
		b.WriteString("spring.data.redis.host=localhost\nspring.data.redis.port=6379\n")
	}
	if has(deps, "data-neo4j") {
		b.WriteString("spring.neo4j.uri=bolt://localhost:7687\nspring.neo4j.authentication.username=neo4j\nspring.neo4j.authentication.password=password\n")
	}
	if has(deps, "actuator") {
		b.WriteString("management.endpoints.web.exposure.include=health,info,metrics\nmanagement.endpoint.health.show-details=always\n")
	}

	// Nothing to add (no relevant deps) — leave the file untouched.
	if b.Len() <= len("\n# --- Added by Sprout ---\n") {
		return nil
	}

	path := filepath.Join(dest, "src", "main", "resources", "application.properties")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(b.String())
	return err
}

func InjectJavaTemplates(dest, packageName, language string, deps []string, hasWeb bool) error {
	if language != "java" {
		return nil // Only supporting Java template injection for this prototype
	}

	parts := strings.Split(packageName, ".")

	pathElements := []string{dest, "src", "main", "java"}
	pathElements = append(pathElements, parts...)
	pkgPath := filepath.Join(pathElements...)

	os.MkdirAll(pkgPath, os.ModePerm)

	// Only inject the REST controller when Spring Web is present, otherwise the
	// generated project won't compile.
	if hasWeb {
		controllerContent := fmt.Sprintf(`package %s;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class HelloController {

    @GetMapping("/api/hello")
    public String sayHello() {
        return "Hello from Sprout! 🌱";
    }
}
`, packageName)
		if err := os.WriteFile(filepath.Join(pkgPath, "HelloController.java"), []byte(controllerContent), 0644); err != nil {
			return err
		}
	}

	for _, d := range deps {
		if d == "data-redis" {
			redisConfigContent := fmt.Sprintf(`package %s;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.data.redis.connection.RedisConnectionFactory;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.data.redis.serializer.StringRedisSerializer;

@Configuration
public class RedisConfig {

    @Bean
    public RedisTemplate<String, Object> redisTemplate(RedisConnectionFactory connectionFactory) {
        RedisTemplate<String, Object> template = new RedisTemplate<>();
        template.setConnectionFactory(connectionFactory);
        // Human-readable string keys (values keep the default serializer).
        template.setKeySerializer(new StringRedisSerializer());
        template.setHashKeySerializer(new StringRedisSerializer());
        template.afterPropertiesSet();
        return template;
    }
}
`, packageName)
			os.WriteFile(filepath.Join(pkgPath, "RedisConfig.java"), []byte(redisConfigContent), 0644)
		}
	}

	return nil
}
