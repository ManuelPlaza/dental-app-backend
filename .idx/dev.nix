{ pkgs, ... }: {
  # Usamos el canal estable de Nix
  channel = "stable-23.11";

  # Aquí instalamos los programas
  packages = [
    pkgs.go
    pkgs.postgresql
    pkgs.lsof
    # --- AGREGADO PARA FLUTTER ---
    pkgs.flutter
    pkgs.dart
    pkgs.cmake       # Necesario para construir la app en Linux
    pkgs.ninja
    pkgs.pkg-config
    # -----------------------------
  ];

  # Variables de entorno
  env = {};

  # Configuración de IDX
  idx = {
    extensions = [
      "golang.go"
      "Dart-Code.flutter"   # <--- Extensión visual de Flutter
      "Dart-Code.dart-code" # <--- Extensión visual de Dart
    ];
    
    # Esto es opcional, previsualizaciones
    previews = {};
    
    # Workspace
    workspace = {
      onCreate = {
        # Configuración inicial (opcional)
      };
    };
  };
}