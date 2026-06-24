package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

func hasSecurity(cfg Config) bool {
	return cfg.ProjectType == "security" || has(cfg.Deps, "security")
}

var dbNameInvalid = regexp.MustCompile(`[^a-z0-9_]`)

// dbName returns a SQL-safe database identifier derived from the project name.
func dbName(cfg Config) string {
	s := strings.ToLower(FolderName(cfg))
	s = strings.ReplaceAll(s, "-", "_")
	s = dbNameInvalid.ReplaceAllString(s, "_")
	if s == "" {
		s = "app"
	}
	if s[0] >= '0' && s[0] <= '9' {
		s = "db_" + s
	}
	return s
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

// InjectDocker writes a multi-stage Dockerfile and a docker-compose.yml that
// wires an `app` service to the backing services it spins up.
func InjectDocker(dest string, cfg Config) error {
	deps := cfg.Deps
	java := cfg.JavaVersion
	if java == "" {
		java = "21"
	}
	db := dbName(cfg)

	var svc strings.Builder
	var depends, appEnv, volumes []string

	switch {
	case has(deps, "postgresql") || (has(deps, "data-jpa") && !has(deps, "mysql") && !has(deps, "h2")):
		svc.WriteString(fmt.Sprintf(postgresBlock, db))
		depends = append(depends, "postgres")
		volumes = append(volumes, "pgdata")
		appEnv = append(appEnv,
			fmt.Sprintf("      DB_URL: jdbc:postgresql://postgres:5432/%s", db),
			"      DB_USERNAME: user",
			"      DB_PASSWORD: password")
	case has(deps, "mysql"):
		svc.WriteString(fmt.Sprintf(mysqlBlock, db))
		depends = append(depends, "mysql")
		volumes = append(volumes, "mysqldata")
		appEnv = append(appEnv,
			fmt.Sprintf("      DB_URL: jdbc:mysql://mysql:3306/%s", db),
			"      DB_USERNAME: user",
			"      DB_PASSWORD: password")
	}

	if has(deps, "data-redis") {
		svc.WriteString(redisBlock)
		depends = append(depends, "redis")
		appEnv = append(appEnv, "      REDIS_HOST: redis")
	}
	if has(deps, "data-neo4j") {
		svc.WriteString(neo4jBlock)
		depends = append(depends, "neo4j")
		appEnv = append(appEnv, "      NEO4J_URI: bolt://neo4j:7687")
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

	var dockerfile string
	if strings.HasPrefix(cfg.BuildTool, "gradle") {
		// Boot's Gradle plugin also emits a *-plain.jar; pick the runnable one.
		dockerfile = fmt.Sprintf(`FROM eclipse-temurin:%s-jdk AS build
WORKDIR /app
COPY . .
RUN chmod +x gradlew && ./gradlew --no-daemon -x test clean bootJar \
 && cp "$(ls build/libs/*.jar | grep -v plain | head -n1)" app.jar

FROM eclipse-temurin:%s-jre
WORKDIR /app
RUN groupadd -r spring && useradd -r -g spring spring
USER spring:spring
COPY --from=build /app/app.jar app.jar
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
RUN groupadd -r spring && useradd -r -g spring spring
USER spring:spring
COPY --from=build /app/target/*.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java","-jar","app.jar"]
`, java, java)
	}

	return os.WriteFile(filepath.Join(dest, "Dockerfile"), []byte(dockerfile), 0644)
}

// InjectProperties wires runtime config. Safe defaults (env-overridable) go in
// application.properties; noisy/unsafe dev toggles go in application-dev.properties
// under the `dev` profile.
func InjectProperties(dest string, cfg Config) error {
	deps := cfg.Deps
	db := dbName(cfg)
	res := filepath.Join(dest, "src", "main", "resources")

	var base, dev strings.Builder

	// Architect Default: Enable Virtual Threads for Java 21+
	javaVer := 21 // Default if empty
	if cfg.JavaVersion != "" {
		fmt.Sscanf(cfg.JavaVersion, "%d", &javaVer)
	}
	if javaVer >= 21 {
		base.WriteString("spring.threads.virtual.enabled=true\n")
	}

	if has(deps, "data-jpa") {
		switch {
		case has(deps, "mysql"):
			base.WriteString(fmt.Sprintf("spring.datasource.url=${DB_URL:jdbc:mysql://localhost:3306/%s}\n", db))
			base.WriteString("spring.datasource.username=${DB_USERNAME:user}\nspring.datasource.password=${DB_PASSWORD:password}\n")
		case has(deps, "h2"):
			base.WriteString(fmt.Sprintf("spring.datasource.url=${DB_URL:jdbc:h2:mem:%s}\n", db))
			base.WriteString("spring.datasource.username=${DB_USERNAME:sa}\nspring.datasource.password=${DB_PASSWORD:}\n")
			dev.WriteString("spring.h2.console.enabled=true\n")
		default: // postgres
			base.WriteString(fmt.Sprintf("spring.datasource.url=${DB_URL:jdbc:postgresql://localhost:5432/%s}\n", db))
			base.WriteString("spring.datasource.username=${DB_USERNAME:user}\nspring.datasource.password=${DB_PASSWORD:password}\n")
		}
		dev.WriteString("spring.jpa.hibernate.ddl-auto=update\nspring.jpa.show-sql=true\n")
	}

	if has(deps, "data-redis") {
		base.WriteString("spring.data.redis.host=${REDIS_HOST:localhost}\nspring.data.redis.port=${REDIS_PORT:6379}\n")
	}
	if has(deps, "data-neo4j") {
		base.WriteString("spring.neo4j.uri=${NEO4J_URI:bolt://localhost:7687}\nspring.neo4j.authentication.username=${NEO4J_USER:neo4j}\nspring.neo4j.authentication.password=${NEO4J_PASSWORD:password}\n")
	}
	if has(deps, "actuator") {
		base.WriteString("management.endpoints.web.exposure.include=health,info\n")
		dev.WriteString("management.endpoints.web.exposure.include=health,info,metrics,env,beans\nmanagement.endpoint.health.show-details=always\n")
	}

	// Activate the dev profile by default when there are dev-only settings.
	// Runtime SPRING_PROFILES_ACTIVE overrides this for prod.
	if dev.Len() > 0 {
		header := "\n# --- Added by Sprout (safe defaults; override via env) ---\nspring.profiles.active=dev\n"
		if err := appendFile(filepath.Join(res, "application.properties"), header+base.String()); err != nil {
			return err
		}
		devContent := "# Development-only settings (active under the 'dev' profile).\n# Do NOT rely on these in production.\n" + dev.String()
		return os.WriteFile(filepath.Join(res, "application-dev.properties"), []byte(devContent), 0644)
	}

	if base.Len() == 0 {
		return nil
	}
	return appendFile(filepath.Join(res, "application.properties"), "\n# --- Added by Sprout (override via env) ---\n"+base.String())
}

func appendFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// InjectSourceTemplates writes sample/config sources for Java or Kotlin.
func InjectSourceTemplates(dest string, cfg Config) error {
	lang := cfg.Language
	if lang != "java" && lang != "kotlin" {
		return nil // groovy templates not supported
	}
	kotlin := lang == "kotlin"
	srcRoot := "java"
	ext := ".java"
	if kotlin {
		srcRoot = "kotlin"
		ext = ".kt"
	}

	parts := strings.Split(cfg.PackageName, ".")
	pkgPath := filepath.Join(append([]string{dest, "src", "main", srcRoot}, parts...)...)
	// Enterprise Package Scaffolding
	webPkg := filepath.Join(pkgPath, "web")
	configPkg := filepath.Join(pkgPath, "config")
	domainPkg := filepath.Join(pkgPath, "domain")
	servicePkg := filepath.Join(pkgPath, "service")
	repoPkg := filepath.Join(pkgPath, "repository")
	dtoPkg := filepath.Join(pkgPath, "dto")
	exceptionPkg := filepath.Join(pkgPath, "exception")

	for _, p := range []string{webPkg, configPkg, domainPkg, servicePkg, repoPkg, dtoPkg, exceptionPkg} {
		if err := os.MkdirAll(p, os.ModePerm); err != nil {
			return err
		}
	}

	// Also ensure testing environment is cleanly separated (Spring Initializr creates src/test/java, but let's mirror the package structure)
	testPkgPath := filepath.Join(append([]string{dest, "src", "test", srcRoot}, parts...)...)
	for _, p := range []string{"web", "service", "repository"} {
		if err := os.MkdirAll(filepath.Join(testPkgPath, p), os.ModePerm); err != nil {
			return err
		}
	}

	if hasWeb(cfg) && !cfg.SkipSamples {
		content := controllerSrc(cfg.PackageName+".web", kotlin)
		if err := os.WriteFile(filepath.Join(webPkg, "HelloController"+ext), []byte(content), 0644); err != nil {
			return err
		}
	}
	// Without a SecurityConfig, Spring Security locks every route — the sample
	// endpoint would 401. Permit it (and the health check) explicitly.
	if hasSecurity(cfg) {
		content := securitySrc(cfg.PackageName+".config", kotlin)
		if err := os.WriteFile(filepath.Join(configPkg, "SecurityConfig"+ext), []byte(content), 0644); err != nil {
			return err
		}
	}
	if has(cfg.Deps, "data-redis") {
		content := redisSrc(cfg.PackageName+".config", kotlin)
		if err := os.WriteFile(filepath.Join(configPkg, "RedisConfig"+ext), []byte(content), 0644); err != nil {
			return err
		}
	}

	InjectDependencies(dest, cfg)

	return nil
}

// InjectDependencies modifies the pom.xml or build.gradle to add OpenAPI and Testcontainers.
func InjectDependencies(dest string, cfg Config) {
	hasWeb := hasWeb(cfg)
	hasDb := has(cfg.Deps, "postgresql") || has(cfg.Deps, "mysql") || has(cfg.Deps, "data-jpa") || has(cfg.Deps, "data-redis")

	if !hasWeb && !hasDb {
		return
	}

	if strings.HasPrefix(cfg.BuildTool, "gradle") {
		content, err := os.ReadFile(filepath.Join(dest, "build.gradle"))
		if err == nil {
			str := string(content)
			inject := ""
			if hasWeb {
				inject += "\timplementation 'org.springdoc:springdoc-openapi-starter-webmvc-ui:2.5.0'\n"
			}
			if hasDb {
				inject += "\ttestImplementation 'org.springframework.boot:spring-boot-testcontainers'\n\ttestImplementation 'org.testcontainers:junit-jupiter'\n"
			}
			str = strings.Replace(str, "dependencies {", "dependencies {\n"+inject, 1)
			os.WriteFile(filepath.Join(dest, "build.gradle"), []byte(str), 0644)
		}
	} else {
		content, err := os.ReadFile(filepath.Join(dest, "pom.xml"))
		if err == nil {
			str := string(content)
			inject := ""
			if hasWeb {
				inject += "\n\t\t<dependency>\n\t\t\t<groupId>org.springdoc</groupId>\n\t\t\t<artifactId>springdoc-openapi-starter-webmvc-ui</artifactId>\n\t\t\t<version>2.5.0</version>\n\t\t</dependency>"
			}
			if hasDb {
				inject += "\n\t\t<dependency>\n\t\t\t<groupId>org.springframework.boot</groupId>\n\t\t\t<artifactId>spring-boot-testcontainers</artifactId>\n\t\t\t<scope>test</scope>\n\t\t</dependency>\n\t\t<dependency>\n\t\t\t<groupId>org.testcontainers</groupId>\n\t\t\t<artifactId>junit-jupiter</artifactId>\n\t\t\t<scope>test</scope>\n\t\t</dependency>"
			}
			str = strings.Replace(str, "</dependencies>", inject+"\n\t</dependencies>", 1)
			os.WriteFile(filepath.Join(dest, "pom.xml"), []byte(str), 0644)
		}
	}
}

func controllerSrc(pkg string, kotlin bool) string {
	if kotlin {
		return fmt.Sprintf(`package %s

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

@RestController
class HelloController {
    @GetMapping("/api/hello")
    fun sayHello(): String = "Hello from Sprout! 🌱"
}
`, pkg)
	}
	return fmt.Sprintf(`package %s;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class HelloController {

    @GetMapping("/api/hello")
    public String sayHello() {
        return "Hello from Sprout! 🌱";
    }
}
`, pkg)
}

func securitySrc(pkg string, kotlin bool) string {
	if kotlin {
		return fmt.Sprintf(`package %s

import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.security.config.Customizer
import org.springframework.security.config.annotation.web.builders.HttpSecurity
import org.springframework.security.web.SecurityFilterChain

@Configuration
class SecurityConfig {
    @Bean
    fun filterChain(http: HttpSecurity): SecurityFilterChain {
        http
            .authorizeHttpRequests {
                it.requestMatchers("/api/hello", "/actuator/health").permitAll()
                  .anyRequest().authenticated()
            }
            .httpBasic(Customizer.withDefaults())
        return http.build()
    }
}
`, pkg)
	}
	return fmt.Sprintf(`package %s;

import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.config.Customizer;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.web.SecurityFilterChain;

@Configuration
public class SecurityConfig {

    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .authorizeHttpRequests(auth -> auth
                .requestMatchers("/api/hello", "/actuator/health").permitAll()
                .anyRequest().authenticated())
            .httpBasic(Customizer.withDefaults());
        return http.build();
    }
}
`, pkg)
}

func redisSrc(pkg string, kotlin bool) string {
	if kotlin {
		return fmt.Sprintf(`package %s

import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.data.redis.connection.RedisConnectionFactory
import org.springframework.data.redis.core.RedisTemplate
import org.springframework.data.redis.serializer.StringRedisSerializer

@Configuration
class RedisConfig {
    @Bean
    fun redisTemplate(connectionFactory: RedisConnectionFactory): RedisTemplate<String, Any> {
        val template = RedisTemplate<String, Any>()
        template.connectionFactory = connectionFactory
        template.keySerializer = StringRedisSerializer()
        template.hashKeySerializer = StringRedisSerializer()
        template.afterPropertiesSet()
        return template
    }
}
`, pkg)
	}
	return fmt.Sprintf(`package %s;

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
`, pkg)
}

// InjectCI generates a GitHub Actions workflow to build and test the generated project.
func InjectCI(dest string, cfg Config) error {
	workflowsDir := filepath.Join(dest, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, os.ModePerm); err != nil {
		return err
	}

	cache := "maven"
	buildCmd := "./mvnw -B package --file pom.xml"
	if strings.HasPrefix(cfg.BuildTool, "gradle") {
		cache = "gradle"
		buildCmd = "./gradlew build"
	}

	javaVersion := cfg.JavaVersion
	if javaVersion == "" {
		javaVersion = "21"
	}

	yaml := fmt.Sprintf(`name: Build and Test

on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up JDK %s
      uses: actions/setup-java@v3
      with:
        java-version: '%s'
        distribution: 'temurin'
        cache: '%s'
    - name: Build with %s
      run: %s
`, javaVersion, javaVersion, cache, cache, buildCmd)

	return os.WriteFile(filepath.Join(workflowsDir, "build.yml"), []byte(yaml), 0644)
}

// InjectK8s generates a base Kubernetes deployment and service for the application.
func InjectK8s(dest string, cfg Config) error {
	k8sDir := filepath.Join(dest, "k8s")
	if err := os.MkdirAll(k8sDir, os.ModePerm); err != nil {
		return err
	}

	appName := dbName(cfg)

	deployYaml := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  labels:
    app: %s
spec:
  replicas: 2
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
        env:
        # Example environment variables; wire these to Secrets in production!
        - name: SPRING_PROFILES_ACTIVE
          value: "prod"
        - name: DB_URL
          value: "jdbc:postgresql://your-db-host:5432/%s"
        - name: DB_USERNAME
          value: "user"
        - name: DB_PASSWORD
          value: "password"
        livenessProbe:
          httpGet:
            path: /actuator/health/liveness
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /actuator/health/readiness
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
`, appName, appName, appName, appName, appName, appName, appName)

	svcYaml := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: %s
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: ClusterIP
`, appName, appName)

	if err := os.WriteFile(filepath.Join(k8sDir, "deployment.yaml"), []byte(deployYaml), 0644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(k8sDir, "service.yaml"), []byte(svcYaml), 0644)
}
