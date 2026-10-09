package com.example.examplemod;

import java.util.function.Function;
import net.minecraft.item.Item;
import net.minecraft.registry.Registries;
import net.minecraft.registry.Registry;
import net.minecraft.registry.RegistryKey;
import net.minecraft.registry.RegistryKeys;
import net.minecraft.util.Identifier;

public final class ModItems {
	public static final Item RUBY = register("ruby", Item::new, new Item.Settings());

	private ModItems() {
	}

	public static Item register(String name, Function<Item.Settings, Item> factory, Item.Settings settings) {
		RegistryKey<Item> key = RegistryKey.of(RegistryKeys.ITEM, Identifier.of(ExampleMod.MOD_ID, name));
		Item item = factory.apply(settings.registryKey(key));
		return Registry.register(Registries.ITEM, key, item);
	}

	public static void initialize() {
		// Loading this class runs the static registrations above.
	}
}
