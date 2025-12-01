{ pkgs, ... }: {
  # Usamos el canal estable de Nix
  channel = "stable-23.11";

  # Aquí instalamos los programas
  packages = [
    pkgs.go
    pkgs.postgresql
    pkgs.lsof
  ];

  # Variables de entorno
  env = {};

  # Configuración de IDX
  idx = {
    extensions = [
      "golang.go"
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