plugins {
    `java-library`
    id("net.neoforged.moddev") version "2.0.112"
}

val modId = providers.gradleProperty("mod_id").get()

version = "1.0.0"
group = "com.example.examplemod"

base {
    archivesName = modId
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

neoForge {
    version = providers.gradleProperty("neo_version").get()

    runs {
        register("client") {
            client()
        }
        register("server") {
            server()
        }
    }

    mods {
        register(modId) {
            sourceSet(sourceSets.main.get())
        }
    }
}
