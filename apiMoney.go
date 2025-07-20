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
  
  _ "modernc.org/sqlite"
  "github.com/gorilla/mux"
)

var db *sql.DB

//GETS

//getEgresos consulta los egresos en la tabla, que sean egresos y luego
//los envia en formati json al navegador.
func getEgresos(w http.ResponseWriter, r *http.Request) {
  //consultamos en la tabla los egresos
  //CAMBIAR USUARIO CUANDO SE HAGA EL LOGIN
  registros, err := getRegistros("egreso", "carlos")
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
    return
  }
  
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slice a json y los enviamos
  //al usuario, valodamos el error
  json.NewEncoder(w).Encode(registros)
}

func getIngresos(w http.ResponseWriter, r *http.Request) {
  //consultamos los movimientos tipo ingreso, validamos el error.
  registros, err := getRegistros("ingreso", "carlos")
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
    return
  }
  
  //Establecemos el header de tipo json
  w.Header().Set("Contenct-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  //Pasamos todos los datos del slice a json y los enviamos
  //al usuario, valodamos el error
  err = json.NewEncoder(w).Encode(registros)
  if err != nil {
    http.Error(w, "error al enviar los datos del getIngresos", http.StatusInternalServerError)
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
  
  total, err := getTotal("egreso", desde, hasta, "carlos")
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
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
  
 total, err := getTotal("ingreso", desde, hasta, "carlos")
  if err != nil {
    writeError(w, "Error en al consultar los registros", err, http.StatusInternalServerError)
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
  
  m, err := getRegistroById(id, "carlos")
  if err != nil {
    writeError(w, "Error en al consultar el registro", err, http.StatusInternalServerError)
    return
  }
  
  //establecemos cabeceras y respondemos con un json.
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(m)
}

//exportFechas exporta a un archivo .json o .csv despendiendo del tipo 
//dado solo los regustris que esten dentro del rango de fechas que se le pase.
//ejm http://10.151.44.98:8080/exportRango?desde=2024-12-04T00:00:00Z&hasta=2024-12-20T00:00:00Z&tipo=json
func exportFechas(w http.ResponseWriter, r *http.Request) {
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
  //obtenemos el tipo de archivo y validamos que solo sean los que maneja
  tipo := string(r.URL.Query().Get("tipo"))
  if tipo != "json" && tipo != "csv" {
    http.Error(w, "El tipo solo puede ser json o csv.", http.StatusBadRequest)
    return
  }
  
  //Consultamos la base de datis, validamos el error.
  registros, err := getRegistrosFechas(desde, hasta, "carlos")
  if err != nil {
    writeError(w, "Error al consultar los regustris en la base de datos.", err, http.StatusInternalServerError)
    return
  }
  
	if tipo == "csv" {
	  //creamos el archivo .csv
    archivo, err := crearArchivo(tipo)
  	defer archivo.Close()
    
    //creamos un writer del archivo para poder escrubirle
    writer := csv.NewWriter(archivo)
    defer writer.Flush()
    
    //Recirremos el slite de movimientos para imprimirlos en cada fila del csv  
    for _, fila := range registros {
      //La funcion movimientoASlice pass cada estructura tupo Movimiento a
      //un slite de string
      err = writer.Write(movimientoASlice(fila))
      if err != nil {
        errorStr := fmt.Sprintf("Error al escribir el el archivo. %v", err)
        http.Error(w, errorStr, http.StatusInternalServerError)
        return
      }
    }
	} else {
	  //creamos el archivo .json
	  archivo, err := crearArchivo(tipo)
  	defer archivo.Close()
  	
  	//Creamos un Encoder del archivo para escribir formato json en el.
  	encoder := json.NewEncoder(archivo)
  	encoder.SetIndent("", "  ")
  	//escribimos todo el slite de movimientos
  	err = encoder.Encode(registros)
  	if err != nil {
  	  http.Error(w, "Error al escribir en el archivo", http.StatusInternalServerError)
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
  var m Registro
  //Decodifico el dato de un json a la variable creada al mismo tiempo que evaluo el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    http.Error(w, "Error al leer el json.", http.StatusBadRequest)
    return
  }
  //validamos que si engresara los campos obligatorios
  err = comprobarInfoRequest(m)
  if err != nil {
    http.Error(w, "Error, datos omitidos en el egreso", http.StatusBadRequest)
    return
  }
  
  //Establesco las variables que se usaran para la manejar los movimientos.
  m.Tipo = "egreso"
  
  //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO registros ( tipo, monto, descripcion, grupo, fecha, usuario ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, m.Usuario)
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
  var m Registro
  //leemos los datos json y los pasamos a las estructura
  //comprobamos el error
  err := json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al leer los datos del json. %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //validamos que si ingresara los campos obligatorios
  err = comprobarInfoRequest(m)
  if err != nil {
    http.Error(w, "Error, datos de movimiento omitidos.", http.StatusBadRequest)
    return
  }
  
  m.Tipo = "ingreso"
  
    //Insertamos los datos en la tabla movimienos de la base de datos
  _, err = db.Exec("INSERT INTO registros ( tipo, monto, descripcion, grupo, fecha, usuario ) VALUES(?, ?, ?, ?, ?, ?)", m.Tipo, m.Monto, m.Descripcion, m.Grupo, m.Fecha, m.Usuario)
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
  var m Registro
  //Pasamos los datos a la variable y comprobamos el error.
  err = json.NewDecoder(r.Body).Decode(&m)
  if err != nil {
    errorStr := fmt.Sprintf("Error al leer los datos del json. %v", err)
    http.Error(w, errorStr, http.StatusBadRequest)
    return
  }
  
  //Actualizamos los datos en la tabla por id y validamos el error.
  _, err = db.Exec("UPDATE registros SET monto = ?, descripcion = ?, grupo = ?, fecha = ? WHERE id = ?", m.Monto, m.Descripcion, m.Grupo, m.Fecha, id)
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
  stmt, err := db.Prepare("DELETE FROM registros WHERE id = ?")
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
  r.HandleFunc("/exportRango", exportFechas).Methods("GET")
  
  
  
  server := http.Server{
    Addr: "10.151.44.98:8080",
    Handler: r,
    WriteTimeout: 10 * time.Second,
    ReadTimeout: 10 * time.Second,
    MaxHeaderBytes: 1 << 20,
  }
  
  log.Println("Listening in http://10.151.44.98:8080...")
  log.Fatal(server.ListenAndServe())
}