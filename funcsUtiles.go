package main

import (
  "fmt"
  "database/sql"
  "strconv"
  "log"
  "os"
  "time"
  "net/http"
  
  _ "modernc.org/sqlite"
)

//Estructura regustro: para la base de datos manejaremos los movimientos positivos y negativos
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
  
  _, err = db.Exec(crearTabla)
  if err != nil {
    log.Fatal("Error creando la tabla", err)
  }
}

func comprobarInfoRequest(m Registro) error{
  if m.Monto == 0 || m.Fecha.IsZero() {
    return fmt.Errorf("Error al ingresar los datos, datos importantes son omitidos")
  }
  return nil
}

//movimientoASlice convierte una escmtructura de tipo Movimiento
//para luego pasarlo a un slite de string.
//Esta funcion es para escribir en el archivo .csv
func movimientoASlice(m Registro) []string {
	return []string{
		strconv.Itoa(m.Id),
		m.Tipo,
		strconv.Itoa(m.Monto),
		m.Descripcion,
		m.Grupo,
		m.Fecha.Format("2006-01-02"),
		m.Usuario,
	}
}

func getRegistros(tipo string, usuario string) (registros []Registro, err error) {
  
  var rows *sql.Rows
  
  if tipo == "todos" {
      //consultamos en la tabla los egresos
    rows, err = db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE usuario = ?", usuario)
  } else {
    rows, err = db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE tipo = ? AND usuario = ?", tipo, usuario)
  }
  //Comprobamos el error
  if err != nil {
    err = fmt.Errorf("Error al leer los datos de la tabla, %v", err)
    return 
  }
  defer rows.Close() //cerramos la base de datos
  
  //Recibiremos la base de datos para luego agregarla a slite.
  //Recorremos cada fila con el for.
  for rows.Next() {
    var m Registro//Variable para escanear los registros
    //Escaneamos cada registo ya que es un for y cada vez escaneamos y comprobamos el error.
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Usuario)
    if err != nil {
      err = fmt.Errorf("Error al escanear en la estructura cada registro, %v", err)
      return
    }
    
    //Almacenamos los datos.
    registros = append(registros, m)
  }
  
  return 
}

func getRegistrosFechas(desde time.Time, hasta time.Time, usuario string) (registros []Registro, err error) {
  //consultamos en la tabla los egresos
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, usuario FROM registros WHERE usuario = ? AND fecha BETWEEN ? AND ?", usuario, desde, hasta)

  //Comprobamos el error
  if err != nil {
    err = fmt.Errorf("Error al leer los datos de la tabla, %v", err)
    return 
  }
  defer rows.Close() //cerramos la base de datos
  
  //Recibiremos la base de datos para luego agregarla a slite.
  //Recorremos cada fila con el for.
  for rows.Next() {
    var m Registro//Variable para escanear los registros
    //Escaneamos cada registo ya que es un for y cada vez escaneamos y comprobamos el error.
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Usuario)
    if err != nil {
      err = fmt.Errorf("Error al escanear en la estructura cada registro, %v", err)
      return
    }
    
    //Almacenamos los datos.
    registros = append(registros, m)
  }
  
  return 
}

func getTotal(tipo string, desde time.Time, hasta time.Time, usuario string) (int, error) {
    //variable para escanear el total
  var total int
  //Consultamos de monto los valores con el tipo "egreso" y los sumamos.
  //asegurqndo con COALESCE que no devuelva nil siempre que no tenga valores
  //entre las fechas dadas. Validamos el error y scaneamos el total.
  err := db.QueryRow("SELECT COALESCE(SUM(monto), 0) FROM registros WHERE tipo = ? AND usuario = ? AND fecha BETWEEN ? AND ?", tipo, usuario, desde, hasta).Scan(&total)
  if err != nil {
    err := fmt.Errorf("Error al consultar y sumar los egresos de la base de datos, %v", err)
    return 0, err
  }
  
  return total, err
}

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



//CAMBIAR A ENVIARLE EL ARCHIVO AL USUARIO
func crearArchivo(tipo string) (archivo *os.File, err error) {
  // Obtener la fecha actual en formato YYYY-MM-DD
	fechaActual := time.Now().Format("2006-01-02")
	//Creamos un archivo de nombre movimiento_FECHAACTUAL
  nombreArchivo := fmt.Sprintf("registros_%s.%s", fechaActual, tipo)
  
  // Crear el archivo nuevo si ni existe y si no existe lo actualiza.
  archivo, err = os.OpenFile(nombreArchivo, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
  if err != nil {
    err = fmt.Errorf("Error al crear al erchivo, %v", err)
    return
  }
  return
}

func writeError(w http.ResponseWriter, s string, err error, status int) {
  errorStr := fmt.Sprintf("Error: %s, Descripcion: %v", s, err)
  http.Error(w, errorStr, status)
}