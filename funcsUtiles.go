package main

import (
  "fmt"
  "database/sql"
  "strconv"
  "strings"
  "log"
  "os"
  "time"
  "net/http"
  "context"
  
  "golang.org/x/crypto/bcrypt"
  "github.com/golang-jwt/jwt"
  _ "modernc.org/sqlite"
)

//Estructura registro: para la base de datos manejaremos los movimientos positivos y negativos
//con la misma estructura. El campo tipo sera el que ayude a identificar si el valor es un egreso
//o un ingreso y le dara la naturaleza al movimiento. En la bd se usara una tabla.
type Registro struct {
  Id int `json:"id"`
  Tipo string `json:"tipo"`
  Monto int `json:"monto"`
  Descripcion string `json:"descripcion"`
  Grupo string `json:"grupo"`
  Fecha time.Time `json:"fecha"`
  Usuario string `json:"usuario"`
}

//Escructura para dar respuesta de los datos. De momebto solo usada en 
//la funcion que exporta para el tipo de archivo csv
type RegistroSimple struct {
  Tipo string `json:"tipo"`
  Monto int `json:"monto"`
  Descripcion string `json:"descripcion"`
  Grupo string `json:"grupo"`
  Fecha time.Time `json:"fecha"`
}

type Usuario struct {
  Nombre string `json:"nombre"`
  Clave string `json:"clave"`
}

//Funcion que crea la base de datos. crea el archivo y
//la inicializa si no existe
func initDB() {
  var err error
  db, err = sql.Open("sqlite", "registros.db")
  if err != nil {
    log.Fatal(err)
  }
  
  crearTabla := `
  CREATE TABLE IF NOT EXISTS registros(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tipo TEXT,
  monto INTEGER,
  descripcion TEXT,
  grupo TEXT,
  fecha DATETIME,
  usuario TEXT
  );`
  
  crearTablaUsuarios := `
  CREATE TABLE IF NOT EXISTS usuarios(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  nombre TEXT UNIQUE NOT NULL,
  clave TEXT NOT NULL
  );`
  
  //crear tabla para registros.
  _, err = db.Exec(crearTabla)
  if err != nil {
    log.Fatal("Error creando la tabla", err)
  }
  
    //crear tabla para usuarios y claves
  _, err = db.Exec(crearTablaUsuarios)
  if err != nil {
    log.Fatal("Error creando la tabla usuarios", err)
  }
}

//comprobarInfoRequest se encarga de comprobar si para un registro los datos
//estan el el formato correcto y si estan completos.
//ToDo: AGREGAR LAS VALIDACIONES DE DATOS PARA DAR MAS SEGURIDAD
func comprobarInfoRequest(m Registro) error{
  if m.Monto == 0 || m.Fecha.IsZero() {
    return fmt.Errorf("Error al ingresar los datos, datos importantes son omitidos")
  }
  return nil
}

//movimientoASlice convierte una escmtructura de tipo Registro
//para luego pasarlo a un slite de string.
//Esta funcion es para escribir en el archivo .csv
func movimientoASlice(m Registro) []string {
	return []string{
		m.Tipo,
		strconv.Itoa(m.Monto),
		m.Descripcion,
		m.Grupo,
		m.Fecha.Format("2006-01-02"),
	}
}

func registrosASimples(registros []Registro) (s []RegistroSimple) {
  for _, m := range registros {
    s = append(s, RegistroSimple{
      m.Tipo,
		  m.Monto,
		  m.Descripcion,
		  m.Grupo,
		  m.Fecha,
    })
  }
  
  return 
}

//getRegistros consulta en la base de datos los registros que coincidan con el
//usuario y tipo de registro dado.
func getRegistros(tipo string, usuario string) (registros []Registro, err error) {
  
  //el puntero que recibira la consulta Query, lo usaremos para escanaer los datos.
  var rows *sql.Rows
  
  if tipo == "todos" {
      //consultamos en la tabla todos los registros
    rows, err = db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE usuario = ?", usuario)
  } else {
    //consultamos en la tabla los registros segun el tipo.
    rows, err = db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE tipo = ? AND usuario = ?", tipo, usuario)
  }
  //Comprobamos el error
  if err != nil {
    err = fmt.Errorf("Error al leer los datos de la tabla, %v", err)
    return 
  }
  defer rows.Close() //cerramos la base de consulta
  
  //Recorremos cada fila con el for.
  for rows.Next() {
    var m Registro//Variable para escanear los registros
    
    //Escaneamos cada registo ya que es un for y cada vez escaneamos y comprobamos el error.
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Usuario)
    if err != nil {
      err = fmt.Errorf("Error al escanear en la estructura cada registro, %v", err)
      return
    }
    
    //Almacenamos los datos para retornarlos en un slite
    registros = append(registros, m)
  }
  
  return 
}

//getRegistrosFechas devuelve los registros que esten dentro de las fechas
//dadas para cada usuario.
func getRegistrosFechas(desde time.Time, hasta time.Time, usuario string) (registros []Registro, err error) {
  //consultamos en la tabla los egresos
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE usuario = ? AND fecha BETWEEN ? AND ?", usuario, desde, hasta)

  //Comprobamos el error
  if err != nil {
    err = fmt.Errorf("Error al leer los datos de la tabla, %v", err)
    return 
  }
  defer rows.Close() //cerramos la consulta
  
  //Recorremos cada fila con el for.
  for rows.Next() {
    var m Registro//Variable para escanear los registros
    //Escaneamos cada registo ya que es un for y cada vez escaneamos y comprobamos el error.
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Usuario)
    if err != nil {
      err = fmt.Errorf("Error al escanear en la estructura cada registro, %v", err)
      return
    }
    
    //Almacenamos los datos para luego retornar el slite
    registros = append(registros, m)
  }
  
  return 
}

//getTotal retorna la suma de cada registro que este dentro del rango dado
//dependiendo del tipo y el usuario.
func getTotal(tipo string, desde time.Time, hasta time.Time, usuario string) (int, error) {
    //variable para escanear el total
  var total int
  //Consultamos cada monto que coincida con el tipo y los sumamos.
  //asegurando con COALESCE que no devuelva nil siempre que no tenga valores
  //entre las fechas dadas. Validamos el error y scaneamos el total.
  err := db.QueryRow("SELECT COALESCE(SUM(monto), 0) FROM registros WHERE tipo = ? AND usuario = ? AND fecha BETWEEN ? AND ?", tipo, usuario, desde, hasta).Scan(&total)
  if err != nil {
    err := fmt.Errorf("Error al consultar y sumar los egresos de la base de datos, %v", err)
    return 0, err
  }
  
  return total, err
}

//getRegistroById retorna un registro segun el id y el usuario.
func getRegistroById(id int, usuario string) (Registro, error) {
  //Estructura para obtener los datos de la base de dato.
  var m Registro
  
  //consultamos por id y validamos el error.
  err := db.QueryRow("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE id = ? AND usuario = ?", id, usuario).Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Usuario)
  if err != nil {
    err := fmt.Errorf("Error al consultar en la base de datos el id ingresado. %v", err)
    return m, err
  }
  
  return m, err
}

//guardarUsuario guarda un usuario y su clave hasheada.
func guardarUsuario(u Usuario) error {
  hash, err := bcrypt.GenerateFromPassword([]byte(u.Clave), bcrypt.DefaultCost)
  if err != nil {
    return err
  }
  
  u.Clave = string(hash)
  
  _, err = db.Exec("INSERT INTO usuarios( nombre, clave ) VALUES( ?, ? )", u.Nombre, u.Clave)
  if err != nil {
    return err
  }
  return nil
}

func comprobarUsuario(u Usuario) error {
  var hashUser string
  err := db.QueryRow("SELECT clave FROM usuarios WHERE nombre = ?", u.Nombre).Scan(&hashUser)
  if err != nil {
    return err
  }
  
  return bcrypt.CompareHashAndPassword([]byte(hashUser), []byte(u.Clave))
}

func crearJWT(nombre string) (string, error) {
  firma := os.Getenv("FRASE")
  
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"nombreUsuario": nombre,
		"exp": time.Now().Add(2 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(firma))
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

func authMiddleware(siguiente http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    
    autorizacion := r.Header.Get("Authorization")
    if !strings.HasPrefix(autorizacion, "Bearer ") {
      http.Error(w, "Falta el token o toke  errado.", http.StatusBadRequest)
      return
    }
    
    tokenString := strings.TrimPrefix(autorizacion, "Bearer ")
    firma := os.Getenv("FRASE")
    
    token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
      
      if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
        return nil, fmt.Errorf("Firma inesperada.")
      }
      
      return []byte(firma), nil
    })
    
    if err != nil || !token.Valid {
      writeError(w, "Token invalido,", err, http.StatusBadRequest)
      return
    }
    
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
      http.Error(w, "Token invalido.", http.StatusBadRequest)
      return
    }
    nombre := claims["nombreUsuario"].(string)
    ctx := context.WithValue(r.Context(), "usuario", nombre)
    
    siguiente.ServeHTTP(w, r.WithContext(ctx))
  })
}

//writeError se encarga de escribir en el responseWriter el error dado.
func writeError(w http.ResponseWriter, s string, err error, status int) {
  errorStr := fmt.Sprintf("Error: %s, Descripcion: %v", s, err)
  http.Error(w, errorStr, status)
}