package example.text

actual fun platformName(): String = "JVM ${System.getProperty("java.version")}"
