# Push ECR

Para compilar la application solo hay que ejecutar el comando

```bash
go build -o output/pushECR
```

Luego agregarlo a alguna carpeta que se quiera usar para almacenar los scripts personalizados

Agregar la carpeta al PATH

Para usarlo hay que configurar un archivo de tipo yml con la siguiete estructura dentro del repositorio que en el que se quiere utilizar

```yaml
profiles:
  dev:
    ecr:
      region:
      account_id:
      repository:
      image_tag:
    docker:
      image_name:
  prod:
    ecr:
      region:
      account_id:
      repository:
      image_tag:
    docker:
      image_name:
```

Para ejecutar el programa tenemos los siguientes flags

### -config
Con este flag definimos que archivo usara al momento de la ejecución por defecto toma el archivo config.yml

Ejemplo:
```shell
pushECR -config deploy.yml
```

### -profile

Con la variable profile se define que configuration se quiere utilizar en la estructura anterior tenemos dev y prod
por defecto usa el ambiente de dev ejemplo de uso

```shell
pushECR -profile dev
```

#### Ejemplo del comando completo

```shell
pushECR -config deploy.yml -profile dev
```