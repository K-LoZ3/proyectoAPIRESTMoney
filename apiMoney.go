package main

import (
  "net/http"
  "time"
  "fmt"
  "database/sql"
  "encoding/json"
  "encoding/csv"
  "log"
  "strconv"
  "os"
  
  _ "modernc.org/sqlite"
  "github.com/gorilla/mux"
)

//Eliminar al finalizar con los handlers
func holaMundo(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Hola Mundo!")
}

//Estructura movimiento: para la base de datos manejaremos los movimientos positivos y negativos
//con la misma estructura. El campo tipo sera el que ayude a identificar si el valor es un egreso
//o un ingreso y le dara la naturaleza al movimiento. En la bd se usara una tabla.
type Movimiento struct {
  Id int `json:"id"`
  Tipo string `json:"tipo"`
  Monto int `json:"monto"`
  Descripcion string `json:"descripcion"`
  Grupo string `json:"grupo"`
  Fecha time.Time `json:"fecha"`
  Creado time.Time `json:"creado"`
}

var db *sql.DB

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

//GETS

//getEgresos consulta los egresos en la tabla, que sean egresos y luego
//los envia en formati json al navegador.
func getEgresos(w http.ResponseWriter, r *http.Request) {
  //consultamos en la tabla los egresos
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos WHERE tipo = ?", "egreso")
  //Comprobamos el error
  if err != nil {
    http.Error(w, "Error al leer los datos de la tabla.", http.StatusBadRequest)
   return 
  }
  defer rows.Close() //cerramos la base de datos
  
  //Creamos el slice para agrupar todos los egresos
  var movimientos []Movimiento
  //recorremos cada fila co  un for.
  for rows.Next() {
    var m Movimiento
    //Escaneamos los datos por cada fila y los pasamos a la estructura
    //validamos el error al mismo tiempo
    if err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado) ; err != nil {
      http.Error(w, "Error al pasar los datos de la tabla a la estructura.", http.StatusBadRequest)
      return
    }
    //agregamls los datos a a la estructura
    movimientos = append(movimientos, m)
  }
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slice a json y los enviamos
  //al usuario, valodamos el error
  err = json.NewEncoder(w).Encode(movimientos)
  if err != nil {
    http.Error(w, "error al enviar los datos del getEgresos", http.StatusInternalServerError)
  }
}

func getIngresos(w http.ResponseWriter, r *http.Request) {
  //consultamos los movimientos tipo ingreso, validamos el error.
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos WHERE tipo = ?", "ingreso")
  if err != nil {
    http.Error(w, "Error al consultar los ingresos el la db.", http.StatusBadRequest)
    return
  }
  defer rows.Close()
  
  //Creo la variable para almacenar los movimientos
  var movimientos []Movimiento
  for rows.Next() {
    var m Movimiento
    //Escaneamos los datos de la cada fila
    err = rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado)
    if err != nil {
      http.Error(w, "Error al scanear los datos de la fila.", http.StatusBadRequest)
      return
    }
    //Agregamos cada fila/structura al slice 
    movimientos = append(movimientos, m)
  }
  
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slice a json y los enviamos
  //al usuario, valodamos el error
  err = json.NewEncoder(w).Encode(movimientos)
  if err != nil {
    http.Error(w, "error al enviar los datos del getEgresos", http.StatusInternalServerError)
  }
}

//getTotalEgresos devuelve el total de egresos dependiendo de las fechas
//que se le pasen, sumara todo entre ellas y devolvera solo la suma
//consulta http://100.69.187.16:8080/totalEgresos?desde=2024-12-20T00:00:00Z&hasta=2024-12-31T00:00:00Z
func getTotalEgresos(w http.ResponseWriter, r *http.Request) {
  //Recibe las fechas y en el formato para time.Time y validamos el error.
  desde, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("desde"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'desde', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  hasta, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("hasta"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'hasta', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  //variable para escanear el total
  var total int
  //Consultamos de monto los valores con el tipo "egreso" y los sumamos.
  //asegurqndo con COALESCE que no devuelva nil siempre que no tenga valores
  //entre las fechas dadas. Validamos el error y scaneamos el total.
  err = db.QueryRow("SELECT COALESCE(SUM(monto), 0) FROM movimientos WHERE tipo = ? AND fecha BETWEEN ? AND ?", "egreso", desde, hasta).Scan(&total)
  if err != nil {
    errorStr := fmt.Sprintf("Error al consultar y sumar los egresos de la base de datos, %v", err)
    http.Error(w, errorStr, http.StatusInternalServerError)
    return
  }
  
  // Devolvemos el total en JSON, lo pasamos a map ya que la funcion Encode
  //necesita un tipo de dato que sea compatiple para codificar.
  jsonTotalEgresos := map[string]int{"total": total}
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(jsonTotalEgresos)
}

//getTotalIngresos devuelve el total de ingresos dependiendo de las fechas
//que se le pasen, sumara todo entre ellas y devolvera solo la suma
//consulta http://100.69.187.16:8080/totalIngresos?desde=2024-12-04T00:00:00Z&hasta=2024-12-20T00:00:00Z
func getTotalIngresos(w http.ResponseWriter, r *http.Request) {
  
  //Convertimos los strings a formato fecha y validamos los errores
  desde, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("desde"))
  if err != nil {
    errorStr := fmt.Sprintf("Error al ingresae la fecha, %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  hasta, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("hasta"))
  if err != nil {
    errorStr := fmt.Sprintf("Error al ingresae la fecha, %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //Creo la variabke que escaneara el valor de la suma de la consulta
  var total int
  //Realizamos la consulta en la tabla, COALESCE(,) asegura que no
  //retornara nil si no hay valores que sumar, validamos el error.
  err = db.QueryRow("SELECT COALESCE(SUM(monto), 0) FROM movimientos WHERE tipo = ? AND fecha BETWEEN ? AND ?", "ingreso", desde, hasta).Scan(&total)
  if err != nil {
    http.Error(w, "Error al consultar y sumar los ingresos.", http.StatusInternalServerError)
    return
  }
  
  // Devolvemos el total en JSON, lo pasamos a map ya que la funcion Encode
  //necesita un tipo de dato que sea compatiple para codificar.
  jsonTotalIngresos := map[string]int{"total": total}
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(jsonTotalIngresos)
}

//getById retorna un moviviento dependiendo solo del id que se pasa como
//variqble en la URL. ejm: http://100.69.187.16:8080/movimiento/10
func getById(w http.ResponseWriter, r *http.Request) {
  //Sacamos la variable.
  //validamos que sea de tipo int
  id, err := strconv.Atoi(mux.Vars(r)["id"])
  if err != nil {
    http.Error(w, "Error en id, se esperaba un numero de tipo int.", http.StatusBadRequest)
    return
  }
  
  //Estructura para obtener los datos de la base de dato.
  var m Movimiento
  
  //consultamos por id y validamos el error.
  err = db.QueryRow("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos WHERE id = ?", id).Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado)
  if err != nil {
    errorStr := fmt.Sprintf("Error al consultar en la base de datos el id ingresado. %v", err)
    http.Error(w, errorStr, http.StatusInternalServerError)
    return
  }
  
  //establecemos cabeceras y respondemos con un json.
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(m)
}

func exportMovimientos(w http.ResponseWriter, r *http.Request) {
  
  //Sacamos la variable.
  tipo := mux.Vars(r)["type"]
  if tipo != "json" && tipo != "csv" {
    http.Error(w, "Error en tipo del archivo, se esperaba un tipo json o csv", http.StatusBadRequest)
    return
  }
  
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos")
  if err != nil {
    http.Error(w, "Error al consultar los regustris en la base de datos.", http.StatusInternalServerError)
    return
  }
  defer rows.Close()
  
  var movimientos []Movimiento
  
  for rows.Next() {
    var m Movimiento
    err := rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado)
    if err != nil {
      errorStr := fmt.Sprintf("Error al escanear en la escmtructura cada moviviento, %v", err)
      http.Error(w, errorStr, http.StatusInternalServerError)
    }
    
    movimientos = append(movimientos, m)
  }
  
  if tipo == "json" {
    archivo, err := os.Create("movimientos.json")
    if err != nil {
      http.Error(w, "Error al crear al erchivo", http.StatusInternalServerError)
      return
    }
    defer archivo.Close()
    
    encoder := json.NewEncoder(archivo)
    encoder.SetIndent("", "  ")
    err = encoder.Encode(movimientos)
    if err != nil {
      http.Error(w, "Error al escribir en el archivo", http.StatusInternalServerError)
    }
  } else {
    archivo, err := os.Create("movimientos.csv")
    if err != nil {
      http.Error(w, "Error al crear al erchivo", http.StatusInternalServerError)
      return
    }
    
    defer archivo.Close()
    
    writer := csv.NewWriter(archivo)
    defer writer.Flush()
    
    for _, fila := range movimientos {
      err = writer.Write(movimientoASlice(fila))
      if err != nil {
        errorStr := fmt.Sprintf("Error al escribir el el archivo. %v", err)
        http.Error(w, errorStr, http.StatusInternalServerError)
        return
      }
    }
  }

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
}

func exportCsvFechas(w http.ResponseWriter, r *http.Request) {
  //Recibe las fechas y en el formato para time.Time y validamos el error.
  desde, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("desde"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'desde', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  hasta, err := time.Parse("2006-01-02T00:00:00Z", r.URL.Query().Get("hasta"))
  if err != nil {
    errorStr := fmt.Sprintf("Error en la fecha ingresada 'hasta', %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  rows, err := db.Query("SELECT id, tipo, monto, descripcion, grupo, fecha, creado FROM movimientos WHERE fecha BETWEEN ? AND ?", desde, hasta)
  if err != nil {
    http.Error(w, "Error al consultar los regustris en la base de datos.", http.StatusInternalServerError)
    return
  }
  defer rows.Close()
  
  var movimientos []Movimiento
  
  for rows.Next() {
    var m Movimiento
    err := rows.Scan(&m.Id, &m.Tipo, &m.Monto, &m.Descripcion, &m.Grupo, &m.Fecha, &m.Creado)
    if err != nil {
      errorStr := fmt.Sprintf("Error al escanear en la escmtructura cada moviviento, %v", err)
      http.Error(w, errorStr, http.StatusInternalServerError)
    }
    
    movimientos = append(movimientos, m)
  }
  
  // Obtener la fecha actual en formato YYYY-MM-DD
	fechaActual := time.Now().Format("2006-01-02")
	nombreArchivo := fmt.Sprintf("movimientos_%s.csv", fechaActual)

	// Crear el archivo nuevo
	archivo, err := os.OpenFile(nombreArchivo, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
  if err != nil {
    http.Error(w, "Error al crear al erchivo", http.StatusInternalServerError)
    return
  }
	defer archivo.Close()
  
  writer := csv.NewWriter(archivo)
  defer writer.Flush()
    
  for _, fila := range movimientos {
    err = writer.Write(movimientoASlice(fila))
    if err != nil {
      errorStr := fmt.Sprintf("Error al escribir el el archivo. %v", err)
      http.Error(w, errorStr, http.StatusInternalServerError)
      return
    }
  }
  
  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
}

//POSTS

//postEgreso agrega un moviviento en la tabla de tipo egreso, se resive con un Json.
//Json ejemplo{"monto": 22,"fecha": "2024-12-05T00:00:00Z"}
func postEgreso(w http.ResponseWriter, r *http.Request) {
  //Creo la variable para almacenar los datos que envia el cliente
  var m Movimiento
  //Decodifico el dato de un json a la variable creada al mismo tiempo que evaluo el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    http.Error(w, "Error al leer el json.", http.StatusBadRequest)
    return
  }
  //validamos que si engresara los campos obligatorios
  err = comprobarMovimiento(m)
  if err != nil {
    http.Error(w, "Error, datos omitidos en el egreso", http.StatusBadRequest)
    return
  }
  
  //Establesco las variables que se usaran para la manejar los movimientos.
  m.Tipo = "egreso"
  m.Creado = time.Now()
  
  //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO movimientos ( tipo, monto, descripcion, grupo, fecha, creado ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, m.Creado)
  //Valido el error al insertar los datos
  if err != nil {
    http.Error(w, "Error al insertar egreso en la tabla.", http.StatusInternalServerError)
    return
  }
  
  //Establesco la cabecera para responder
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
  //Convertimos la estructura a json y enviamos los datos
  //comprobamos el error al pasarlos
  err = json.NewEncoder(w).Encode(m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al escribir el json con los datos que se ingresaron. %v", err)
    http.Error(w, errorStr, http.StatusInternalServerError)
  }
}

//postIngreso agrega a la base de datos un movimiento con el tipo ingreso
func postIngreso(w http.ResponseWriter, r *http.Request) {
  var m Movimiento
  //leemos los datos json y los pasamos a las estructura
  //comprobamos el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al leer los datos del json. %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //validamos que si ingresara los campos obligatorios
  err = comprobarMovimiento(m)
  if err != nil {
    http.Error(w, "Error, datos de movimiento omitidos.", http.StatusBadRequest)
    return
  }
  
  m.Tipo = "ingreso"
  m.Creado = time.Now()
  
    //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO movimientos ( tipo, monto, descripcion, grupo, fecha, creado ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, m.Creado)
  //Valido el error al insertar los datos
  if err != nil {
    http.Error(w, "Error al insertar ingreso en la tabla.", http.StatusInternalServerError)
    return
  }
  
  //Establesco la cabecera para responder
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusCreated)
  //Convertimos la estructura a json y enviamos los datos
  //comprobamos el error al pasarlos
  err = json.NewEncoder(w).Encode(m)
  if err != nil {
    http.Error(w, "Error al escribir el json con los datos que se ingresaron.", http.StatusInternalServerError)
  }
}

//PUTS

//putById actualiza un registro en la tabla segun el id que se pase como
//variable por URL con los datos tipo json a travez del body. De momento se
//asume que el cliente envia los datos completos.
//ejm http://100.69.187.16:8080/movimiento/9
// {"monto": 333, "grupo": "nuevo"}
//ToDo: LOS DATOS OMITIDOS DEJARLOS CON EL MISMO VALOR.
func putById(w http.ResponseWriter, r *http.Request) {
  //Extraemos la el id de la URL y aseguramos que sea un int.
  id, err := strconv.Atoi(mux.Vars(r)["id"])
  if err != nil {
    http.Error(w, "Error en id, se esperaba un numero de tipo int.", http.StatusBadRequest)
    return
  }
  
  //Variable para extraer los datos a actualizar.
  //De momenro los actualiza asumiendo que pasa todos los datos.
  var m Movimiento
  //Pasamos los datos a la variable y comprobamos el error.
  err = json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al leer los datos del json. %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //Actualizamos los datos en la tabla por id y validamos el error.
  _, err = db.Exec("UPDATE movimientos SET monto = ?, descripcion = ?, grupo = ?, fecha = ? WHERE id = ?", m.Monto, m.Descripcion, m.Grupo, m.Fecha, id)
  if err != nil {
    errorStr := fmt.Sprintf("Error al actualizar el registro en la base de datos con el id ingresado. %v", err)
    http.Error(w, errorStr, http.StatusInternalServerError)
    return
  }
  
  //establecemos cabeceras y respondemos con un json.
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(m)
}

//DELETES

//deleteById elimina un registro segun el id ingresado.
func deleteById(w http.ResponseWriter, r *http.Request) {
  //Extraemos la el id de la URL y aseguramos que sea un int.
  id, err := strconv.Atoi(mux.Vars(r)["id"])
  if err != nil {
    http.Error(w, "Error en id, se esperaba un numero de tipo int.", http.StatusBadRequest)
    return
  }
  
  //preparamos la instruccion para sqlite.
  stmt, err := db.Prepare("DELETE FROM movimientos WHERE id = ?")
  if err != nil {
    http.Error(w, "Error preparando SQL", http.StatusInternalServerError)
    return
  }
  //cerramos la base de datos.
  defer stmt.Close()
  
  //Ejecutamos la instruccion para eliminar el regustri.
  res, err := stmt.Exec(id)
  if err != nil {
    http.Error(w, "Error ejecutando DELETE", http.StatusInternalServerError)
    return
  }
  
  //Validamos que si elimine el regustro
  filas, err := res.RowsAffected()
  if err != nil || filas == 0 {
    http.Error(w, "No se eliminó ningún registro", http.StatusNotFound)
    return
  }
  
  //respondemos al usuario.
  w.WriteHeader(http.StatusOK)
  fmt.Fprintf(w, "Movimiento con ID %s eliminado correctamente", id)
}

func main() {
  initDB()
  defer db.Close()
  r := mux.NewRouter()
  
  r.HandleFunc("/egreso", getEgresos).Methods("GET")
  r.HandleFunc("/ingreso", getIngresos).Methods("GET")
  r.HandleFunc("/ingreso", postIngreso).Methods("POST")
  r.HandleFunc("/egreso", postEgreso).Methods("POST")
  r.HandleFunc("/totalEgresos", getTotalEgresos).Methods("GET")
  r.HandleFunc("/totalIngresos", getTotalIngresos).Methods("GET")
  r.HandleFunc("/movimiento/{id}", getById).Methods("GET")
  r.HandleFunc("/movimiento/{id}", putById).Methods("PUT")
  r.HandleFunc("/movimiento/{id}", deleteById).Methods("DELETE")
  r.HandleFunc("/export/{type}", exportMovimientos).Methods("GET")
  r.HandleFunc("/csvRango", exportCsvFechas).Methods("GET")
  
  
  
  server := http.Server{
    Addr: "100.69.187.16:8080",
    Handler: r,
    WriteTimeout: 10 * time.Second,
    ReadTimeout: 10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  
  log.Println("Listening in http://100.69.187.16:8080...")
  log.Fatal(server.ListenAndServe())
}