{ pkgs, ... }: {
channel = "stable-24.05"; # <--- La versión que te pedí poner
  # ... (Tu código de channel y packages igual que antes) ...

  packages = [
    pkgs.go
    pkgs.postgresql
    pkgs.lsof
    # --- CONFIGURACIÓN DE FLUTTER QUE FUERZA UNA INSTALACIÓN LIMPIA ---
    (pkgs.flutter.override { enableStableChannel = true; })
    pkgs.dart
    pkgs.cmake
    pkgs.ninja
    pkgs.pkg-config
  ];

  env = {};

  # --- CONFIGURACIÓN DE VISTA PREVIA (EL CÓDIGO QUE FALTABA) ---
  idx = {
    extensions = [
      "golang.go"
      "Dart-Code.flutter"
      "Dart-Code.dart-code"
    ];
    
    previews = { # <--- AÑADIR ESTE BLOQUE
        enable = true;
        previews = {
            web = {
              # --- AQUÍ ESTÁ EL CAMBIO CRÍTICO: AGREGAR EL 'cd' ---
                command = ["bash" "-c" "cd dental_frontend && flutter run --machine -d web-server --web-hostname 0.0.0.0 --web-port $PORT"];
                manager = "flutter";
            };
        };
    };
    
    workspace = {
      onCreate = {};
    };
  };
}