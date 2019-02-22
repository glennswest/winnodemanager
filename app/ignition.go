package main;
  
import(
            "os"
            "fmt"
            "net/url"
            "strings"
            "io"
            "io/ioutil"
            "path/filepath"
            "github.com/tidwall/gjson"
            "net/http"
        )

// Examples
//{
//  "ignition": { "version": "2.2.0" },
//  "storage": {
//    "files": [{
//      "filesystem": "root",
//      "path": "/foo/bar",
//      "mode": 420,
//      "contents": { "source": "data:,example%20file%0A" }
//    }]
//  }
//}
//{
//  "ignition": { "version": "2.2.0" },
//  "storage": {
//    "files": [{
//      "filesystem": "root",
//      "path": "/foo/bar",
//      "mode": 420,
//      "contents": {
//        "source": "http://example.com/asset",
//        "verification": { "hash": "sha512-0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" }
//      }
//    }]
//  }
//}


func main(){
  parse_ignition_file("./ignition-test.json");
}

func parse_ignition_string(tc string) int {
	version := gjson.Get(tc, "ignition.version");
        if (version.String() == ""){
           fmt.Printf("Invalid file");
           return(-1);
           }
        result := gjson.Get(tc,"storage.files");
        files := result.Array();
        for _,tfile := range files {
            tpath := gjson.Get(tfile.String(),"path").String();
            tmode := gjson.Get(tfile.String(),"mode").Int();
            tdata := gjson.Get(tfile.String(),"contents.source").String();
            idx := strings.Index(tdata,":")+1;
            thetype := tdata[:idx];
            fmt.Printf("path: %s type: %s mode %o\n",tpath,thetype,tmode);
            tdir := filepath.Dir(tpath);
            fmt.Printf("%s\n",tdir);
            os.MkdirAll(tdir, os.ModePerm);
            fmt.Printf("Type: path: %s type: %s mode %o\n",tpath,thetype,tmode);
            switch thetype {
               case "data:":
                    untc,_ := url.QueryUnescape(tdata[idx+1:]);
                    td := []byte(untc);
                    err := ioutil.WriteFile(tpath, td, os.FileMode(tmode));
                    if (err != nil){
                       fmt.Printf("Failed to Write %s: %s\n",tpath,err);
                       }
               case "http:","https:":
                    err := downloadfile(tpath,tdata);
                    if (err != nil){
                       fmt.Printf("Download Failed: %s - %s\n",tpath,err);
                       }
               default:
                  fmt.Printf("Invalid Type: path: %s type: %s mode %o\n",tpath,thetype,tmode);
               }
                 
            }
        return(0);
}

func parse_ignition_file(thefilepath string) int {

    b, err :=ioutil.ReadFile(thefilepath);
    if err != nil {
      fmt.Print(err);
      return 0;
      }
    content := string(b);
    result := parse_ignition_string(content);
    return(result);

}


// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadfile(filepath string, url string) error {

    // Get the data
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Create the file
    out, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer out.Close()

    // Write the body to file
    _, err = io.Copy(out, resp.Body)
    return err
}

