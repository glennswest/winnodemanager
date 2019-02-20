package main;

import(
	    "fmt"
	    "os/exec"
			"strings"
	)


func main(){
  thever := GetWinVersion();
	fmt.Printf("Windows Version = %s\n",thever);
  check_windows_prereq();
}

func check_windows_prereq(){
		reset_needed := 0;
		reset_needed =+ check_ipv6();
		reset_needed =+ check_hyperv();
		reset_needed =+ check_name();
		if (reset_needed > 0){
 			PsReset();
 			}
}

func PsReset(){
	fmt.Printf("Reseting\n");
}

func check_ipv6() int {
	thenamespace := "HKLM:\\SYSTEM\\CurrentControlSet\\Services\\Tcpip6\\Parameters";
	thevalue := GetPsRegValue(thenamespace,"DisabledComponents");
	if (thevalue == ""){ // Its not disabled}
	    // Lets Disable it
			fmt.Printf("Disable IPV6\n");
			SetPSRegValue(thenamespace,"DisabledComponents","0xffffffff","DWord");
      return(1);
		}
	return(0);
}

func check_hyperv() int {
	thevalue := GetPsInstalled("Hyper-V");
	if (thevalue == "Installed"){
		 // Disable HyberV
		 cmd := "bcdedit.exe /set hypervisorlaunchtype off";
		 result := powershell(cmd);
		 fmt.Printf("%s\n%s\n",cmd,result);
		 return(1);
	   }
	return(0);
}

func check_name() int {
	hostname := GetPsHostName();
	fmt.Printf("Hostname: %s\n",hostname);
	return(0);
}

func GetLineArray(theresult string ,thelinenum int ) [] string {
	thelines := lines(theresult);
	output := standardizeSpaces(thelines[thelinenum]);
	varvals := strings.Split(output," ");
	return(varvals);
}

func GetPsHostName() string {
	cmd := "$env:COMPUTERNAME";
	result := powershell(cmd);
	fmt.Printf("%s\n%s\n",cmd,result);
	return(result);
}

func GetPsInstalled(thepackage string) string {
	cmd := "Get-WindowsFeature -Name '" + thepackage + "'";
	result := powershell(cmd);
	varvals := GetLineArray(result,3);
	return(varvals[4]);
}

func GetPsRegValue(thenamespace string,thevaluename string) string {
	cmd := "Get-ItemProperty -Path " + thenamespace + " -Name '" + thevaluename + "' -ErrorAction SilentlyContinue";
	result := powershell(cmd);
	if (len(result) == 0){
		return "";
	  }
	varvals := GetLineArray(result,2);
	return(varvals[2]);
}

func SetPSRegValue(thenamespace string, thevaluename string, thevalue string, thetype string){
	thecmd := "New-ItemProperty -Path " + thenamespace + " -Name '" + thevaluename + "' -Value '" + thevalue + "' -PropertyType '" + thetype + "'";
  result := powershell(thecmd);
	fmt.Printf("SetPSRegValue: %s\n", result);
}
func GetWinVersion() string {
// Major  Minor  Build  Revision
// -----  -----  -----  --------
// 10     0      17134  0
	result := powershell("[System.Environment]::OSVersion.Version");
  thelines := lines(result);

	verstr := standardizeSpaces(thelines[3]);
	va := strings.Split(verstr," ");
	ver := va[0] + "." + va[1] + "." + va[2] + "." + va[3];
	return(ver);
}


func lines(theval string) [] string {
	 values := strings.Split(strings.Replace(theval, "\r\n", "\n", -1), "\n");
	 return(values);
}

func standardizeSpaces(s string) string {
    return strings.Join(strings.Fields(s), " ")
}

func powershell(thecmd string) string {
	theargs := strings.Split(thecmd," ");
	c,err := exec.Command("powershell", theargs...).CombinedOutput();
	cmd := string(c);

	if  err != nil {
	    return("");
    } else {
	    return(cmd);
    }
}
