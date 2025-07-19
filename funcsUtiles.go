package main

import (
  "fmt"
  "database/sql"
  "strconv"
  "log"
  "os"
  "time"
  
  _ "modernc.org/sqlite"
)

//Funcion que crea la base de datos. crea el archivo y
//la inicializa si no existe
func initDB() {
  var err error
  db, err = sql.Open("sqlite", "movimientos.db")
  if err != nil {
    log.Fatal(err)
  }
  
  crearTabla := `
  CREATE TABLE IF NOT EXISTS movimientos(
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  tipo TEXT,
  monto INTEGER,
  descripcion TEXT,
  grupo TEXT,
  fecha DATETIME,
  creado DATETIME
  );`
  
  _, err = db.Exec(crearTabla)
  if err != nil {
    log.Fatal("Error creando la tabla", err)
  }
}

//comprobarMovimiento recibe la estructura para validar si se ingrresaron los datos
//obligatorios. Devuelve un error si falta  datos.
func comprobarMovimiento(m Movimiento) error{
  if m.Monto == 0 || m.Fecha.IsZero() {
    return fmt.Errorf("Error al ingresar los datos, datos importantes son omitidos")
  }
  return nil
}

//movimientoASlice convierte una escmtructura de tipo Movimiento
//para luego pasarlo a un slite de string.
//Esta funcion es para escribir en el archivo .csv
func movimientoASlice(m Movimiento) []string {
	return []string{
		strconv.Itoa(m.Id),
		m.Tipo,
		strconv.Itoa(m.Monto),
		m.Descripcion,
		m.Grupo,
		m.Fecha.Format("2006-01-02"),
		m.Creado.Format("2006-01-02"),
	}
}

//dbRowsAMovimientos recibe un pintero de la consulta a la base datos.
//Esta funcioen recorre y almacena en en slite cada uno de los regustros.
//Para luego devorler el slite y el error.
func dbRowsAMovimientos(rows *sql.Rows) (movimientos []Movimiento, err error) {
  //Recibiremos la base de datos para luego agregarla a slite.
  //Recorremos cada fila con el for.
  for rows.Next() {
    var m Movimiento//Variable para escanear los registros
    //Escaneamos cada registo ya que es un for y cada vez escaneamos y comprobamos el error.
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado)
    if err != nil {
      err = fmt.Errorf("Error al escanear en la escmtructura cada moviviento, %v", err)
      return
    }
    
    //Almacenamos los datos.
    movimientos = append(movimientos, m)
  }
  
  return 
}

func crearArchivo(tipo string) (archivo *os.File, err error) {
  // Obtener la fecha actual en formato YYYY-MM-DD
	fechaActual := time.Now().Format("2006-01-02")
	//Creamos un archivo de nombre movimiento_FECHAACTUAL
  nombreArchivo := fmt.Sprintf("movimientos_%s.%s", fechaActual, tipo)
  
  // Crear el archivo nuevo si ni existe y si no existe lo actualiza.
  archivo, err = os.OpenFile(nombreArchivo, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
  if err != nil {
    err = fmt.Errorf("Error al crear al erchivo, %v", err)
    return
  }
  return
}