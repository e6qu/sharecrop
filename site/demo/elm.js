(function(scope){
'use strict';

function F(arity, fun, wrapper) {
  wrapper.a = arity;
  wrapper.f = fun;
  return wrapper;
}

function F2(fun) {
  return F(2, fun, function(a) { return function(b) { return fun(a,b); }; })
}
function F3(fun) {
  return F(3, fun, function(a) {
    return function(b) { return function(c) { return fun(a, b, c); }; };
  });
}
function F4(fun) {
  return F(4, fun, function(a) { return function(b) { return function(c) {
    return function(d) { return fun(a, b, c, d); }; }; };
  });
}
function F5(fun) {
  return F(5, fun, function(a) { return function(b) { return function(c) {
    return function(d) { return function(e) { return fun(a, b, c, d, e); }; }; }; };
  });
}
function F6(fun) {
  return F(6, fun, function(a) { return function(b) { return function(c) {
    return function(d) { return function(e) { return function(f) {
    return fun(a, b, c, d, e, f); }; }; }; }; };
  });
}
function F7(fun) {
  return F(7, fun, function(a) { return function(b) { return function(c) {
    return function(d) { return function(e) { return function(f) {
    return function(g) { return fun(a, b, c, d, e, f, g); }; }; }; }; }; };
  });
}
function F8(fun) {
  return F(8, fun, function(a) { return function(b) { return function(c) {
    return function(d) { return function(e) { return function(f) {
    return function(g) { return function(h) {
    return fun(a, b, c, d, e, f, g, h); }; }; }; }; }; }; };
  });
}
function F9(fun) {
  return F(9, fun, function(a) { return function(b) { return function(c) {
    return function(d) { return function(e) { return function(f) {
    return function(g) { return function(h) { return function(i) {
    return fun(a, b, c, d, e, f, g, h, i); }; }; }; }; }; }; }; };
  });
}

function A2(fun, a, b) {
  return fun.a === 2 ? fun.f(a, b) : fun(a)(b);
}
function A3(fun, a, b, c) {
  return fun.a === 3 ? fun.f(a, b, c) : fun(a)(b)(c);
}
function A4(fun, a, b, c, d) {
  return fun.a === 4 ? fun.f(a, b, c, d) : fun(a)(b)(c)(d);
}
function A5(fun, a, b, c, d, e) {
  return fun.a === 5 ? fun.f(a, b, c, d, e) : fun(a)(b)(c)(d)(e);
}
function A6(fun, a, b, c, d, e, f) {
  return fun.a === 6 ? fun.f(a, b, c, d, e, f) : fun(a)(b)(c)(d)(e)(f);
}
function A7(fun, a, b, c, d, e, f, g) {
  return fun.a === 7 ? fun.f(a, b, c, d, e, f, g) : fun(a)(b)(c)(d)(e)(f)(g);
}
function A8(fun, a, b, c, d, e, f, g, h) {
  return fun.a === 8 ? fun.f(a, b, c, d, e, f, g, h) : fun(a)(b)(c)(d)(e)(f)(g)(h);
}
function A9(fun, a, b, c, d, e, f, g, h, i) {
  return fun.a === 9 ? fun.f(a, b, c, d, e, f, g, h, i) : fun(a)(b)(c)(d)(e)(f)(g)(h)(i);
}

console.warn('Compiled in DEV mode. Follow the advice at https://elm-lang.org/0.19.1/optimize for better performance and smaller assets.');


var _List_Nil_UNUSED = { $: 0 };
var _List_Nil = { $: '[]' };

function _List_Cons_UNUSED(hd, tl) { return { $: 1, a: hd, b: tl }; }
function _List_Cons(hd, tl) { return { $: '::', a: hd, b: tl }; }


var _List_cons = F2(_List_Cons);

function _List_fromArray(arr)
{
	var out = _List_Nil;
	for (var i = arr.length; i--; )
	{
		out = _List_Cons(arr[i], out);
	}
	return out;
}

function _List_toArray(xs)
{
	for (var out = []; xs.b; xs = xs.b) // WHILE_CONS
	{
		out.push(xs.a);
	}
	return out;
}

var _List_map2 = F3(function(f, xs, ys)
{
	for (var arr = []; xs.b && ys.b; xs = xs.b, ys = ys.b) // WHILE_CONSES
	{
		arr.push(A2(f, xs.a, ys.a));
	}
	return _List_fromArray(arr);
});

var _List_map3 = F4(function(f, xs, ys, zs)
{
	for (var arr = []; xs.b && ys.b && zs.b; xs = xs.b, ys = ys.b, zs = zs.b) // WHILE_CONSES
	{
		arr.push(A3(f, xs.a, ys.a, zs.a));
	}
	return _List_fromArray(arr);
});

var _List_map4 = F5(function(f, ws, xs, ys, zs)
{
	for (var arr = []; ws.b && xs.b && ys.b && zs.b; ws = ws.b, xs = xs.b, ys = ys.b, zs = zs.b) // WHILE_CONSES
	{
		arr.push(A4(f, ws.a, xs.a, ys.a, zs.a));
	}
	return _List_fromArray(arr);
});

var _List_map5 = F6(function(f, vs, ws, xs, ys, zs)
{
	for (var arr = []; vs.b && ws.b && xs.b && ys.b && zs.b; vs = vs.b, ws = ws.b, xs = xs.b, ys = ys.b, zs = zs.b) // WHILE_CONSES
	{
		arr.push(A5(f, vs.a, ws.a, xs.a, ys.a, zs.a));
	}
	return _List_fromArray(arr);
});

var _List_sortBy = F2(function(f, xs)
{
	return _List_fromArray(_List_toArray(xs).sort(function(a, b) {
		return _Utils_cmp(f(a), f(b));
	}));
});

var _List_sortWith = F2(function(f, xs)
{
	return _List_fromArray(_List_toArray(xs).sort(function(a, b) {
		var ord = A2(f, a, b);
		return ord === $elm$core$Basics$EQ ? 0 : ord === $elm$core$Basics$LT ? -1 : 1;
	}));
});



var _JsArray_empty = [];

function _JsArray_singleton(value)
{
    return [value];
}

function _JsArray_length(array)
{
    return array.length;
}

var _JsArray_initialize = F3(function(size, offset, func)
{
    var result = new Array(size);

    for (var i = 0; i < size; i++)
    {
        result[i] = func(offset + i);
    }

    return result;
});

var _JsArray_initializeFromList = F2(function (max, ls)
{
    var result = new Array(max);

    for (var i = 0; i < max && ls.b; i++)
    {
        result[i] = ls.a;
        ls = ls.b;
    }

    result.length = i;
    return _Utils_Tuple2(result, ls);
});

var _JsArray_unsafeGet = F2(function(index, array)
{
    return array[index];
});

var _JsArray_unsafeSet = F3(function(index, value, array)
{
    var length = array.length;
    var result = new Array(length);

    for (var i = 0; i < length; i++)
    {
        result[i] = array[i];
    }

    result[index] = value;
    return result;
});

var _JsArray_push = F2(function(value, array)
{
    var length = array.length;
    var result = new Array(length + 1);

    for (var i = 0; i < length; i++)
    {
        result[i] = array[i];
    }

    result[length] = value;
    return result;
});

var _JsArray_foldl = F3(function(func, acc, array)
{
    var length = array.length;

    for (var i = 0; i < length; i++)
    {
        acc = A2(func, array[i], acc);
    }

    return acc;
});

var _JsArray_foldr = F3(function(func, acc, array)
{
    for (var i = array.length - 1; i >= 0; i--)
    {
        acc = A2(func, array[i], acc);
    }

    return acc;
});

var _JsArray_map = F2(function(func, array)
{
    var length = array.length;
    var result = new Array(length);

    for (var i = 0; i < length; i++)
    {
        result[i] = func(array[i]);
    }

    return result;
});

var _JsArray_indexedMap = F3(function(func, offset, array)
{
    var length = array.length;
    var result = new Array(length);

    for (var i = 0; i < length; i++)
    {
        result[i] = A2(func, offset + i, array[i]);
    }

    return result;
});

var _JsArray_slice = F3(function(from, to, array)
{
    return array.slice(from, to);
});

var _JsArray_appendN = F3(function(n, dest, source)
{
    var destLen = dest.length;
    var itemsToCopy = n - destLen;

    if (itemsToCopy > source.length)
    {
        itemsToCopy = source.length;
    }

    var size = destLen + itemsToCopy;
    var result = new Array(size);

    for (var i = 0; i < destLen; i++)
    {
        result[i] = dest[i];
    }

    for (var i = 0; i < itemsToCopy; i++)
    {
        result[i + destLen] = source[i];
    }

    return result;
});



// LOG

var _Debug_log_UNUSED = F2(function(tag, value)
{
	return value;
});

var _Debug_log = F2(function(tag, value)
{
	console.log(tag + ': ' + _Debug_toString(value));
	return value;
});


// TODOS

function _Debug_todo(moduleName, region)
{
	return function(message) {
		_Debug_crash(8, moduleName, region, message);
	};
}

function _Debug_todoCase(moduleName, region, value)
{
	return function(message) {
		_Debug_crash(9, moduleName, region, value, message);
	};
}


// TO STRING

function _Debug_toString_UNUSED(value)
{
	return '<internals>';
}

function _Debug_toString(value)
{
	return _Debug_toAnsiString(false, value);
}

function _Debug_toAnsiString(ansi, value)
{
	if (typeof value === 'function')
	{
		return _Debug_internalColor(ansi, '<function>');
	}

	if (typeof value === 'boolean')
	{
		return _Debug_ctorColor(ansi, value ? 'True' : 'False');
	}

	if (typeof value === 'number')
	{
		return _Debug_numberColor(ansi, value + '');
	}

	if (value instanceof String)
	{
		return _Debug_charColor(ansi, "'" + _Debug_addSlashes(value, true) + "'");
	}

	if (typeof value === 'string')
	{
		return _Debug_stringColor(ansi, '"' + _Debug_addSlashes(value, false) + '"');
	}

	if (typeof value === 'object' && '$' in value)
	{
		var tag = value.$;

		if (typeof tag === 'number')
		{
			return _Debug_internalColor(ansi, '<internals>');
		}

		if (tag[0] === '#')
		{
			var output = [];
			for (var k in value)
			{
				if (k === '$') continue;
				output.push(_Debug_toAnsiString(ansi, value[k]));
			}
			return '(' + output.join(',') + ')';
		}

		if (tag === 'Set_elm_builtin')
		{
			return _Debug_ctorColor(ansi, 'Set')
				+ _Debug_fadeColor(ansi, '.fromList') + ' '
				+ _Debug_toAnsiString(ansi, $elm$core$Set$toList(value));
		}

		if (tag === 'RBNode_elm_builtin' || tag === 'RBEmpty_elm_builtin')
		{
			return _Debug_ctorColor(ansi, 'Dict')
				+ _Debug_fadeColor(ansi, '.fromList') + ' '
				+ _Debug_toAnsiString(ansi, $elm$core$Dict$toList(value));
		}

		if (tag === 'Array_elm_builtin')
		{
			return _Debug_ctorColor(ansi, 'Array')
				+ _Debug_fadeColor(ansi, '.fromList') + ' '
				+ _Debug_toAnsiString(ansi, $elm$core$Array$toList(value));
		}

		if (tag === '::' || tag === '[]')
		{
			var output = '[';

			value.b && (output += _Debug_toAnsiString(ansi, value.a), value = value.b)

			for (; value.b; value = value.b) // WHILE_CONS
			{
				output += ',' + _Debug_toAnsiString(ansi, value.a);
			}
			return output + ']';
		}

		var output = '';
		for (var i in value)
		{
			if (i === '$') continue;
			var str = _Debug_toAnsiString(ansi, value[i]);
			var c0 = str[0];
			var parenless = c0 === '{' || c0 === '(' || c0 === '[' || c0 === '<' || c0 === '"' || str.indexOf(' ') < 0;
			output += ' ' + (parenless ? str : '(' + str + ')');
		}
		return _Debug_ctorColor(ansi, tag) + output;
	}

	if (typeof DataView === 'function' && value instanceof DataView)
	{
		return _Debug_stringColor(ansi, '<' + value.byteLength + ' bytes>');
	}

	if (typeof File !== 'undefined' && value instanceof File)
	{
		return _Debug_internalColor(ansi, '<' + value.name + '>');
	}

	if (typeof value === 'object')
	{
		var output = [];
		for (var key in value)
		{
			var field = key[0] === '_' ? key.slice(1) : key;
			output.push(_Debug_fadeColor(ansi, field) + ' = ' + _Debug_toAnsiString(ansi, value[key]));
		}
		if (output.length === 0)
		{
			return '{}';
		}
		return '{ ' + output.join(', ') + ' }';
	}

	return _Debug_internalColor(ansi, '<internals>');
}

function _Debug_addSlashes(str, isChar)
{
	var s = str
		.replace(/\\/g, '\\\\')
		.replace(/\n/g, '\\n')
		.replace(/\t/g, '\\t')
		.replace(/\r/g, '\\r')
		.replace(/\v/g, '\\v')
		.replace(/\0/g, '\\0');

	if (isChar)
	{
		return s.replace(/\'/g, '\\\'');
	}
	else
	{
		return s.replace(/\"/g, '\\"');
	}
}

function _Debug_ctorColor(ansi, string)
{
	return ansi ? '\x1b[96m' + string + '\x1b[0m' : string;
}

function _Debug_numberColor(ansi, string)
{
	return ansi ? '\x1b[95m' + string + '\x1b[0m' : string;
}

function _Debug_stringColor(ansi, string)
{
	return ansi ? '\x1b[93m' + string + '\x1b[0m' : string;
}

function _Debug_charColor(ansi, string)
{
	return ansi ? '\x1b[92m' + string + '\x1b[0m' : string;
}

function _Debug_fadeColor(ansi, string)
{
	return ansi ? '\x1b[37m' + string + '\x1b[0m' : string;
}

function _Debug_internalColor(ansi, string)
{
	return ansi ? '\x1b[36m' + string + '\x1b[0m' : string;
}

function _Debug_toHexDigit(n)
{
	return String.fromCharCode(n < 10 ? 48 + n : 55 + n);
}


// CRASH


function _Debug_crash_UNUSED(identifier)
{
	throw new Error('https://github.com/elm/core/blob/1.0.0/hints/' + identifier + '.md');
}


function _Debug_crash(identifier, fact1, fact2, fact3, fact4)
{
	switch(identifier)
	{
		case 0:
			throw new Error('What node should I take over? In JavaScript I need something like:\n\n    Elm.Main.init({\n        node: document.getElementById("elm-node")\n    })\n\nYou need to do this with any Browser.sandbox or Browser.element program.');

		case 1:
			throw new Error('Browser.application programs cannot handle URLs like this:\n\n    ' + document.location.href + '\n\nWhat is the root? The root of your file system? Try looking at this program with `elm reactor` or some other server.');

		case 2:
			var jsonErrorString = fact1;
			throw new Error('Problem with the flags given to your Elm program on initialization.\n\n' + jsonErrorString);

		case 3:
			var portName = fact1;
			throw new Error('There can only be one port named `' + portName + '`, but your program has multiple.');

		case 4:
			var portName = fact1;
			var problem = fact2;
			throw new Error('Trying to send an unexpected type of value through port `' + portName + '`:\n' + problem);

		case 5:
			throw new Error('Trying to use `(==)` on functions.\nThere is no way to know if functions are "the same" in the Elm sense.\nRead more about this at https://package.elm-lang.org/packages/elm/core/latest/Basics#== which describes why it is this way and what the better version will look like.');

		case 6:
			var moduleName = fact1;
			throw new Error('Your page is loading multiple Elm scripts with a module named ' + moduleName + '. Maybe a duplicate script is getting loaded accidentally? If not, rename one of them so I know which is which!');

		case 8:
			var moduleName = fact1;
			var region = fact2;
			var message = fact3;
			throw new Error('TODO in module `' + moduleName + '` ' + _Debug_regionToString(region) + '\n\n' + message);

		case 9:
			var moduleName = fact1;
			var region = fact2;
			var value = fact3;
			var message = fact4;
			throw new Error(
				'TODO in module `' + moduleName + '` from the `case` expression '
				+ _Debug_regionToString(region) + '\n\nIt received the following value:\n\n    '
				+ _Debug_toString(value).replace('\n', '\n    ')
				+ '\n\nBut the branch that handles it says:\n\n    ' + message.replace('\n', '\n    ')
			);

		case 10:
			throw new Error('Bug in https://github.com/elm/virtual-dom/issues');

		case 11:
			throw new Error('Cannot perform mod 0. Division by zero error.');
	}
}

function _Debug_regionToString(region)
{
	if (region.start.line === region.end.line)
	{
		return 'on line ' + region.start.line;
	}
	return 'on lines ' + region.start.line + ' through ' + region.end.line;
}



// EQUALITY

function _Utils_eq(x, y)
{
	for (
		var pair, stack = [], isEqual = _Utils_eqHelp(x, y, 0, stack);
		isEqual && (pair = stack.pop());
		isEqual = _Utils_eqHelp(pair.a, pair.b, 0, stack)
		)
	{}

	return isEqual;
}

function _Utils_eqHelp(x, y, depth, stack)
{
	if (x === y)
	{
		return true;
	}

	if (typeof x !== 'object' || x === null || y === null)
	{
		typeof x === 'function' && _Debug_crash(5);
		return false;
	}

	if (depth > 100)
	{
		stack.push(_Utils_Tuple2(x,y));
		return true;
	}

	/**/
	if (x.$ === 'Set_elm_builtin')
	{
		x = $elm$core$Set$toList(x);
		y = $elm$core$Set$toList(y);
	}
	if (x.$ === 'RBNode_elm_builtin' || x.$ === 'RBEmpty_elm_builtin')
	{
		x = $elm$core$Dict$toList(x);
		y = $elm$core$Dict$toList(y);
	}
	//*/

	/**_UNUSED/
	if (x.$ < 0)
	{
		x = $elm$core$Dict$toList(x);
		y = $elm$core$Dict$toList(y);
	}
	//*/

	for (var key in x)
	{
		if (!_Utils_eqHelp(x[key], y[key], depth + 1, stack))
		{
			return false;
		}
	}
	return true;
}

var _Utils_equal = F2(_Utils_eq);
var _Utils_notEqual = F2(function(a, b) { return !_Utils_eq(a,b); });



// COMPARISONS

// Code in Generate/JavaScript.hs, Basics.js, and List.js depends on
// the particular integer values assigned to LT, EQ, and GT.

function _Utils_cmp(x, y, ord)
{
	if (typeof x !== 'object')
	{
		return x === y ? /*EQ*/ 0 : x < y ? /*LT*/ -1 : /*GT*/ 1;
	}

	/**/
	if (x instanceof String)
	{
		var a = x.valueOf();
		var b = y.valueOf();
		return a === b ? 0 : a < b ? -1 : 1;
	}
	//*/

	/**_UNUSED/
	if (typeof x.$ === 'undefined')
	//*/
	/**/
	if (x.$[0] === '#')
	//*/
	{
		return (ord = _Utils_cmp(x.a, y.a))
			? ord
			: (ord = _Utils_cmp(x.b, y.b))
				? ord
				: _Utils_cmp(x.c, y.c);
	}

	// traverse conses until end of a list or a mismatch
	for (; x.b && y.b && !(ord = _Utils_cmp(x.a, y.a)); x = x.b, y = y.b) {} // WHILE_CONSES
	return ord || (x.b ? /*GT*/ 1 : y.b ? /*LT*/ -1 : /*EQ*/ 0);
}

var _Utils_lt = F2(function(a, b) { return _Utils_cmp(a, b) < 0; });
var _Utils_le = F2(function(a, b) { return _Utils_cmp(a, b) < 1; });
var _Utils_gt = F2(function(a, b) { return _Utils_cmp(a, b) > 0; });
var _Utils_ge = F2(function(a, b) { return _Utils_cmp(a, b) >= 0; });

var _Utils_compare = F2(function(x, y)
{
	var n = _Utils_cmp(x, y);
	return n < 0 ? $elm$core$Basics$LT : n ? $elm$core$Basics$GT : $elm$core$Basics$EQ;
});


// COMMON VALUES

var _Utils_Tuple0_UNUSED = 0;
var _Utils_Tuple0 = { $: '#0' };

function _Utils_Tuple2_UNUSED(a, b) { return { a: a, b: b }; }
function _Utils_Tuple2(a, b) { return { $: '#2', a: a, b: b }; }

function _Utils_Tuple3_UNUSED(a, b, c) { return { a: a, b: b, c: c }; }
function _Utils_Tuple3(a, b, c) { return { $: '#3', a: a, b: b, c: c }; }

function _Utils_chr_UNUSED(c) { return c; }
function _Utils_chr(c) { return new String(c); }


// RECORDS

function _Utils_update(oldRecord, updatedFields)
{
	var newRecord = {};

	for (var key in oldRecord)
	{
		newRecord[key] = oldRecord[key];
	}

	for (var key in updatedFields)
	{
		newRecord[key] = updatedFields[key];
	}

	return newRecord;
}


// APPEND

var _Utils_append = F2(_Utils_ap);

function _Utils_ap(xs, ys)
{
	// append Strings
	if (typeof xs === 'string')
	{
		return xs + ys;
	}

	// append Lists
	if (!xs.b)
	{
		return ys;
	}
	var root = _List_Cons(xs.a, ys);
	xs = xs.b
	for (var curr = root; xs.b; xs = xs.b) // WHILE_CONS
	{
		curr = curr.b = _List_Cons(xs.a, ys);
	}
	return root;
}



// MATH

var _Basics_add = F2(function(a, b) { return a + b; });
var _Basics_sub = F2(function(a, b) { return a - b; });
var _Basics_mul = F2(function(a, b) { return a * b; });
var _Basics_fdiv = F2(function(a, b) { return a / b; });
var _Basics_idiv = F2(function(a, b) { return (a / b) | 0; });
var _Basics_pow = F2(Math.pow);

var _Basics_remainderBy = F2(function(b, a) { return a % b; });

// https://www.microsoft.com/en-us/research/wp-content/uploads/2016/02/divmodnote-letter.pdf
var _Basics_modBy = F2(function(modulus, x)
{
	var answer = x % modulus;
	return modulus === 0
		? _Debug_crash(11)
		:
	((answer > 0 && modulus < 0) || (answer < 0 && modulus > 0))
		? answer + modulus
		: answer;
});


// TRIGONOMETRY

var _Basics_pi = Math.PI;
var _Basics_e = Math.E;
var _Basics_cos = Math.cos;
var _Basics_sin = Math.sin;
var _Basics_tan = Math.tan;
var _Basics_acos = Math.acos;
var _Basics_asin = Math.asin;
var _Basics_atan = Math.atan;
var _Basics_atan2 = F2(Math.atan2);


// MORE MATH

function _Basics_toFloat(x) { return x; }
function _Basics_truncate(n) { return n | 0; }
function _Basics_isInfinite(n) { return n === Infinity || n === -Infinity; }

var _Basics_ceiling = Math.ceil;
var _Basics_floor = Math.floor;
var _Basics_round = Math.round;
var _Basics_sqrt = Math.sqrt;
var _Basics_log = Math.log;
var _Basics_isNaN = isNaN;


// BOOLEANS

function _Basics_not(bool) { return !bool; }
var _Basics_and = F2(function(a, b) { return a && b; });
var _Basics_or  = F2(function(a, b) { return a || b; });
var _Basics_xor = F2(function(a, b) { return a !== b; });



var _String_cons = F2(function(chr, str)
{
	return chr + str;
});

function _String_uncons(string)
{
	var word = string.charCodeAt(0);
	return !isNaN(word)
		? $elm$core$Maybe$Just(
			0xD800 <= word && word <= 0xDBFF
				? _Utils_Tuple2(_Utils_chr(string[0] + string[1]), string.slice(2))
				: _Utils_Tuple2(_Utils_chr(string[0]), string.slice(1))
		)
		: $elm$core$Maybe$Nothing;
}

var _String_append = F2(function(a, b)
{
	return a + b;
});

function _String_length(str)
{
	return str.length;
}

var _String_map = F2(function(func, string)
{
	var len = string.length;
	var array = new Array(len);
	var i = 0;
	while (i < len)
	{
		var word = string.charCodeAt(i);
		if (0xD800 <= word && word <= 0xDBFF)
		{
			array[i] = func(_Utils_chr(string[i] + string[i+1]));
			i += 2;
			continue;
		}
		array[i] = func(_Utils_chr(string[i]));
		i++;
	}
	return array.join('');
});

var _String_filter = F2(function(isGood, str)
{
	var arr = [];
	var len = str.length;
	var i = 0;
	while (i < len)
	{
		var char = str[i];
		var word = str.charCodeAt(i);
		i++;
		if (0xD800 <= word && word <= 0xDBFF)
		{
			char += str[i];
			i++;
		}

		if (isGood(_Utils_chr(char)))
		{
			arr.push(char);
		}
	}
	return arr.join('');
});

function _String_reverse(str)
{
	var len = str.length;
	var arr = new Array(len);
	var i = 0;
	while (i < len)
	{
		var word = str.charCodeAt(i);
		if (0xD800 <= word && word <= 0xDBFF)
		{
			arr[len - i] = str[i + 1];
			i++;
			arr[len - i] = str[i - 1];
			i++;
		}
		else
		{
			arr[len - i] = str[i];
			i++;
		}
	}
	return arr.join('');
}

var _String_foldl = F3(function(func, state, string)
{
	var len = string.length;
	var i = 0;
	while (i < len)
	{
		var char = string[i];
		var word = string.charCodeAt(i);
		i++;
		if (0xD800 <= word && word <= 0xDBFF)
		{
			char += string[i];
			i++;
		}
		state = A2(func, _Utils_chr(char), state);
	}
	return state;
});

var _String_foldr = F3(function(func, state, string)
{
	var i = string.length;
	while (i--)
	{
		var char = string[i];
		var word = string.charCodeAt(i);
		if (0xDC00 <= word && word <= 0xDFFF)
		{
			i--;
			char = string[i] + char;
		}
		state = A2(func, _Utils_chr(char), state);
	}
	return state;
});

var _String_split = F2(function(sep, str)
{
	return str.split(sep);
});

var _String_join = F2(function(sep, strs)
{
	return strs.join(sep);
});

var _String_slice = F3(function(start, end, str) {
	return str.slice(start, end);
});

function _String_trim(str)
{
	return str.trim();
}

function _String_trimLeft(str)
{
	return str.replace(/^\s+/, '');
}

function _String_trimRight(str)
{
	return str.replace(/\s+$/, '');
}

function _String_words(str)
{
	return _List_fromArray(str.trim().split(/\s+/g));
}

function _String_lines(str)
{
	return _List_fromArray(str.split(/\r\n|\r|\n/g));
}

function _String_toUpper(str)
{
	return str.toUpperCase();
}

function _String_toLower(str)
{
	return str.toLowerCase();
}

var _String_any = F2(function(isGood, string)
{
	var i = string.length;
	while (i--)
	{
		var char = string[i];
		var word = string.charCodeAt(i);
		if (0xDC00 <= word && word <= 0xDFFF)
		{
			i--;
			char = string[i] + char;
		}
		if (isGood(_Utils_chr(char)))
		{
			return true;
		}
	}
	return false;
});

var _String_all = F2(function(isGood, string)
{
	var i = string.length;
	while (i--)
	{
		var char = string[i];
		var word = string.charCodeAt(i);
		if (0xDC00 <= word && word <= 0xDFFF)
		{
			i--;
			char = string[i] + char;
		}
		if (!isGood(_Utils_chr(char)))
		{
			return false;
		}
	}
	return true;
});

var _String_contains = F2(function(sub, str)
{
	return str.indexOf(sub) > -1;
});

var _String_startsWith = F2(function(sub, str)
{
	return str.indexOf(sub) === 0;
});

var _String_endsWith = F2(function(sub, str)
{
	return str.length >= sub.length &&
		str.lastIndexOf(sub) === str.length - sub.length;
});

var _String_indexes = F2(function(sub, str)
{
	var subLen = sub.length;

	if (subLen < 1)
	{
		return _List_Nil;
	}

	var i = 0;
	var is = [];

	while ((i = str.indexOf(sub, i)) > -1)
	{
		is.push(i);
		i = i + subLen;
	}

	return _List_fromArray(is);
});


// TO STRING

function _String_fromNumber(number)
{
	return number + '';
}


// INT CONVERSIONS

function _String_toInt(str)
{
	var total = 0;
	var code0 = str.charCodeAt(0);
	var start = code0 == 0x2B /* + */ || code0 == 0x2D /* - */ ? 1 : 0;

	for (var i = start; i < str.length; ++i)
	{
		var code = str.charCodeAt(i);
		if (code < 0x30 || 0x39 < code)
		{
			return $elm$core$Maybe$Nothing;
		}
		total = 10 * total + code - 0x30;
	}

	return i == start
		? $elm$core$Maybe$Nothing
		: $elm$core$Maybe$Just(code0 == 0x2D ? -total : total);
}


// FLOAT CONVERSIONS

function _String_toFloat(s)
{
	// check if it is a hex, octal, or binary number
	if (s.length === 0 || /[\sxbo]/.test(s))
	{
		return $elm$core$Maybe$Nothing;
	}
	var n = +s;
	// faster isNaN check
	return n === n ? $elm$core$Maybe$Just(n) : $elm$core$Maybe$Nothing;
}

function _String_fromList(chars)
{
	return _List_toArray(chars).join('');
}




function _Char_toCode(char)
{
	var code = char.charCodeAt(0);
	if (0xD800 <= code && code <= 0xDBFF)
	{
		return (code - 0xD800) * 0x400 + char.charCodeAt(1) - 0xDC00 + 0x10000
	}
	return code;
}

function _Char_fromCode(code)
{
	return _Utils_chr(
		(code < 0 || 0x10FFFF < code)
			? '\uFFFD'
			:
		(code <= 0xFFFF)
			? String.fromCharCode(code)
			:
		(code -= 0x10000,
			String.fromCharCode(Math.floor(code / 0x400) + 0xD800, code % 0x400 + 0xDC00)
		)
	);
}

function _Char_toUpper(char)
{
	return _Utils_chr(char.toUpperCase());
}

function _Char_toLower(char)
{
	return _Utils_chr(char.toLowerCase());
}

function _Char_toLocaleUpper(char)
{
	return _Utils_chr(char.toLocaleUpperCase());
}

function _Char_toLocaleLower(char)
{
	return _Utils_chr(char.toLocaleLowerCase());
}



/**/
function _Json_errorToString(error)
{
	return $elm$json$Json$Decode$errorToString(error);
}
//*/


// CORE DECODERS

function _Json_succeed(msg)
{
	return {
		$: 0,
		a: msg
	};
}

function _Json_fail(msg)
{
	return {
		$: 1,
		a: msg
	};
}

function _Json_decodePrim(decoder)
{
	return { $: 2, b: decoder };
}

var _Json_decodeInt = _Json_decodePrim(function(value) {
	return (typeof value !== 'number')
		? _Json_expecting('an INT', value)
		:
	(-2147483647 < value && value < 2147483647 && (value | 0) === value)
		? $elm$core$Result$Ok(value)
		:
	(isFinite(value) && !(value % 1))
		? $elm$core$Result$Ok(value)
		: _Json_expecting('an INT', value);
});

var _Json_decodeBool = _Json_decodePrim(function(value) {
	return (typeof value === 'boolean')
		? $elm$core$Result$Ok(value)
		: _Json_expecting('a BOOL', value);
});

var _Json_decodeFloat = _Json_decodePrim(function(value) {
	return (typeof value === 'number')
		? $elm$core$Result$Ok(value)
		: _Json_expecting('a FLOAT', value);
});

var _Json_decodeValue = _Json_decodePrim(function(value) {
	return $elm$core$Result$Ok(_Json_wrap(value));
});

var _Json_decodeString = _Json_decodePrim(function(value) {
	return (typeof value === 'string')
		? $elm$core$Result$Ok(value)
		: (value instanceof String)
			? $elm$core$Result$Ok(value + '')
			: _Json_expecting('a STRING', value);
});

function _Json_decodeList(decoder) { return { $: 3, b: decoder }; }
function _Json_decodeArray(decoder) { return { $: 4, b: decoder }; }

function _Json_decodeNull(value) { return { $: 5, c: value }; }

var _Json_decodeField = F2(function(field, decoder)
{
	return {
		$: 6,
		d: field,
		b: decoder
	};
});

var _Json_decodeIndex = F2(function(index, decoder)
{
	return {
		$: 7,
		e: index,
		b: decoder
	};
});

function _Json_decodeKeyValuePairs(decoder)
{
	return {
		$: 8,
		b: decoder
	};
}

function _Json_mapMany(f, decoders)
{
	return {
		$: 9,
		f: f,
		g: decoders
	};
}

var _Json_andThen = F2(function(callback, decoder)
{
	return {
		$: 10,
		b: decoder,
		h: callback
	};
});

function _Json_oneOf(decoders)
{
	return {
		$: 11,
		g: decoders
	};
}


// DECODING OBJECTS

var _Json_map1 = F2(function(f, d1)
{
	return _Json_mapMany(f, [d1]);
});

var _Json_map2 = F3(function(f, d1, d2)
{
	return _Json_mapMany(f, [d1, d2]);
});

var _Json_map3 = F4(function(f, d1, d2, d3)
{
	return _Json_mapMany(f, [d1, d2, d3]);
});

var _Json_map4 = F5(function(f, d1, d2, d3, d4)
{
	return _Json_mapMany(f, [d1, d2, d3, d4]);
});

var _Json_map5 = F6(function(f, d1, d2, d3, d4, d5)
{
	return _Json_mapMany(f, [d1, d2, d3, d4, d5]);
});

var _Json_map6 = F7(function(f, d1, d2, d3, d4, d5, d6)
{
	return _Json_mapMany(f, [d1, d2, d3, d4, d5, d6]);
});

var _Json_map7 = F8(function(f, d1, d2, d3, d4, d5, d6, d7)
{
	return _Json_mapMany(f, [d1, d2, d3, d4, d5, d6, d7]);
});

var _Json_map8 = F9(function(f, d1, d2, d3, d4, d5, d6, d7, d8)
{
	return _Json_mapMany(f, [d1, d2, d3, d4, d5, d6, d7, d8]);
});


// DECODE

var _Json_runOnString = F2(function(decoder, string)
{
	try
	{
		var value = JSON.parse(string);
		return _Json_runHelp(decoder, value);
	}
	catch (e)
	{
		return $elm$core$Result$Err(A2($elm$json$Json$Decode$Failure, 'This is not valid JSON! ' + e.message, _Json_wrap(string)));
	}
});

var _Json_run = F2(function(decoder, value)
{
	return _Json_runHelp(decoder, _Json_unwrap(value));
});

function _Json_runHelp(decoder, value)
{
	switch (decoder.$)
	{
		case 2:
			return decoder.b(value);

		case 5:
			return (value === null)
				? $elm$core$Result$Ok(decoder.c)
				: _Json_expecting('null', value);

		case 3:
			if (!_Json_isArray(value))
			{
				return _Json_expecting('a LIST', value);
			}
			return _Json_runArrayDecoder(decoder.b, value, _List_fromArray);

		case 4:
			if (!_Json_isArray(value))
			{
				return _Json_expecting('an ARRAY', value);
			}
			return _Json_runArrayDecoder(decoder.b, value, _Json_toElmArray);

		case 6:
			var field = decoder.d;
			if (typeof value !== 'object' || value === null || !(field in value))
			{
				return _Json_expecting('an OBJECT with a field named `' + field + '`', value);
			}
			var result = _Json_runHelp(decoder.b, value[field]);
			return ($elm$core$Result$isOk(result)) ? result : $elm$core$Result$Err(A2($elm$json$Json$Decode$Field, field, result.a));

		case 7:
			var index = decoder.e;
			if (!_Json_isArray(value))
			{
				return _Json_expecting('an ARRAY', value);
			}
			if (index >= value.length)
			{
				return _Json_expecting('a LONGER array. Need index ' + index + ' but only see ' + value.length + ' entries', value);
			}
			var result = _Json_runHelp(decoder.b, value[index]);
			return ($elm$core$Result$isOk(result)) ? result : $elm$core$Result$Err(A2($elm$json$Json$Decode$Index, index, result.a));

		case 8:
			if (typeof value !== 'object' || value === null || _Json_isArray(value))
			{
				return _Json_expecting('an OBJECT', value);
			}

			var keyValuePairs = _List_Nil;
			// TODO test perf of Object.keys and switch when support is good enough
			for (var key in value)
			{
				if (value.hasOwnProperty(key))
				{
					var result = _Json_runHelp(decoder.b, value[key]);
					if (!$elm$core$Result$isOk(result))
					{
						return $elm$core$Result$Err(A2($elm$json$Json$Decode$Field, key, result.a));
					}
					keyValuePairs = _List_Cons(_Utils_Tuple2(key, result.a), keyValuePairs);
				}
			}
			return $elm$core$Result$Ok($elm$core$List$reverse(keyValuePairs));

		case 9:
			var answer = decoder.f;
			var decoders = decoder.g;
			for (var i = 0; i < decoders.length; i++)
			{
				var result = _Json_runHelp(decoders[i], value);
				if (!$elm$core$Result$isOk(result))
				{
					return result;
				}
				answer = answer(result.a);
			}
			return $elm$core$Result$Ok(answer);

		case 10:
			var result = _Json_runHelp(decoder.b, value);
			return (!$elm$core$Result$isOk(result))
				? result
				: _Json_runHelp(decoder.h(result.a), value);

		case 11:
			var errors = _List_Nil;
			for (var temp = decoder.g; temp.b; temp = temp.b) // WHILE_CONS
			{
				var result = _Json_runHelp(temp.a, value);
				if ($elm$core$Result$isOk(result))
				{
					return result;
				}
				errors = _List_Cons(result.a, errors);
			}
			return $elm$core$Result$Err($elm$json$Json$Decode$OneOf($elm$core$List$reverse(errors)));

		case 1:
			return $elm$core$Result$Err(A2($elm$json$Json$Decode$Failure, decoder.a, _Json_wrap(value)));

		case 0:
			return $elm$core$Result$Ok(decoder.a);
	}
}

function _Json_runArrayDecoder(decoder, value, toElmValue)
{
	var len = value.length;
	var array = new Array(len);
	for (var i = 0; i < len; i++)
	{
		var result = _Json_runHelp(decoder, value[i]);
		if (!$elm$core$Result$isOk(result))
		{
			return $elm$core$Result$Err(A2($elm$json$Json$Decode$Index, i, result.a));
		}
		array[i] = result.a;
	}
	return $elm$core$Result$Ok(toElmValue(array));
}

function _Json_isArray(value)
{
	return Array.isArray(value) || (typeof FileList !== 'undefined' && value instanceof FileList);
}

function _Json_toElmArray(array)
{
	return A2($elm$core$Array$initialize, array.length, function(i) { return array[i]; });
}

function _Json_expecting(type, value)
{
	return $elm$core$Result$Err(A2($elm$json$Json$Decode$Failure, 'Expecting ' + type, _Json_wrap(value)));
}


// EQUALITY

function _Json_equality(x, y)
{
	if (x === y)
	{
		return true;
	}

	if (x.$ !== y.$)
	{
		return false;
	}

	switch (x.$)
	{
		case 0:
		case 1:
			return x.a === y.a;

		case 2:
			return x.b === y.b;

		case 5:
			return x.c === y.c;

		case 3:
		case 4:
		case 8:
			return _Json_equality(x.b, y.b);

		case 6:
			return x.d === y.d && _Json_equality(x.b, y.b);

		case 7:
			return x.e === y.e && _Json_equality(x.b, y.b);

		case 9:
			return x.f === y.f && _Json_listEquality(x.g, y.g);

		case 10:
			return x.h === y.h && _Json_equality(x.b, y.b);

		case 11:
			return _Json_listEquality(x.g, y.g);
	}
}

function _Json_listEquality(aDecoders, bDecoders)
{
	var len = aDecoders.length;
	if (len !== bDecoders.length)
	{
		return false;
	}
	for (var i = 0; i < len; i++)
	{
		if (!_Json_equality(aDecoders[i], bDecoders[i]))
		{
			return false;
		}
	}
	return true;
}


// ENCODE

var _Json_encode = F2(function(indentLevel, value)
{
	return JSON.stringify(_Json_unwrap(value), null, indentLevel) + '';
});

function _Json_wrap(value) { return { $: 0, a: value }; }
function _Json_unwrap(value) { return value.a; }

function _Json_wrap_UNUSED(value) { return value; }
function _Json_unwrap_UNUSED(value) { return value; }

function _Json_emptyArray() { return []; }
function _Json_emptyObject() { return {}; }

var _Json_addField = F3(function(key, value, object)
{
	object[key] = _Json_unwrap(value);
	return object;
});

function _Json_addEntry(func)
{
	return F2(function(entry, array)
	{
		array.push(_Json_unwrap(func(entry)));
		return array;
	});
}

var _Json_encodeNull = _Json_wrap(null);



// TASKS

function _Scheduler_succeed(value)
{
	return {
		$: 0,
		a: value
	};
}

function _Scheduler_fail(error)
{
	return {
		$: 1,
		a: error
	};
}

function _Scheduler_binding(callback)
{
	return {
		$: 2,
		b: callback,
		c: null
	};
}

var _Scheduler_andThen = F2(function(callback, task)
{
	return {
		$: 3,
		b: callback,
		d: task
	};
});

var _Scheduler_onError = F2(function(callback, task)
{
	return {
		$: 4,
		b: callback,
		d: task
	};
});

function _Scheduler_receive(callback)
{
	return {
		$: 5,
		b: callback
	};
}


// PROCESSES

var _Scheduler_guid = 0;

function _Scheduler_rawSpawn(task)
{
	var proc = {
		$: 0,
		e: _Scheduler_guid++,
		f: task,
		g: null,
		h: []
	};

	_Scheduler_enqueue(proc);

	return proc;
}

function _Scheduler_spawn(task)
{
	return _Scheduler_binding(function(callback) {
		callback(_Scheduler_succeed(_Scheduler_rawSpawn(task)));
	});
}

function _Scheduler_rawSend(proc, msg)
{
	proc.h.push(msg);
	_Scheduler_enqueue(proc);
}

var _Scheduler_send = F2(function(proc, msg)
{
	return _Scheduler_binding(function(callback) {
		_Scheduler_rawSend(proc, msg);
		callback(_Scheduler_succeed(_Utils_Tuple0));
	});
});

function _Scheduler_kill(proc)
{
	return _Scheduler_binding(function(callback) {
		var task = proc.f;
		if (task.$ === 2 && task.c)
		{
			task.c();
		}

		proc.f = null;

		callback(_Scheduler_succeed(_Utils_Tuple0));
	});
}


/* STEP PROCESSES

type alias Process =
  { $ : tag
  , id : unique_id
  , root : Task
  , stack : null | { $: SUCCEED | FAIL, a: callback, b: stack }
  , mailbox : [msg]
  }

*/


var _Scheduler_working = false;
var _Scheduler_queue = [];


function _Scheduler_enqueue(proc)
{
	_Scheduler_queue.push(proc);
	if (_Scheduler_working)
	{
		return;
	}
	_Scheduler_working = true;
	while (proc = _Scheduler_queue.shift())
	{
		_Scheduler_step(proc);
	}
	_Scheduler_working = false;
}


function _Scheduler_step(proc)
{
	while (proc.f)
	{
		var rootTag = proc.f.$;
		if (rootTag === 0 || rootTag === 1)
		{
			while (proc.g && proc.g.$ !== rootTag)
			{
				proc.g = proc.g.i;
			}
			if (!proc.g)
			{
				return;
			}
			proc.f = proc.g.b(proc.f.a);
			proc.g = proc.g.i;
		}
		else if (rootTag === 2)
		{
			proc.f.c = proc.f.b(function(newRoot) {
				proc.f = newRoot;
				_Scheduler_enqueue(proc);
			});
			return;
		}
		else if (rootTag === 5)
		{
			if (proc.h.length === 0)
			{
				return;
			}
			proc.f = proc.f.b(proc.h.shift());
		}
		else // if (rootTag === 3 || rootTag === 4)
		{
			proc.g = {
				$: rootTag === 3 ? 0 : 1,
				b: proc.f.b,
				i: proc.g
			};
			proc.f = proc.f.d;
		}
	}
}



function _Process_sleep(time)
{
	return _Scheduler_binding(function(callback) {
		var id = setTimeout(function() {
			callback(_Scheduler_succeed(_Utils_Tuple0));
		}, time);

		return function() { clearTimeout(id); };
	});
}




// PROGRAMS


var _Platform_worker = F4(function(impl, flagDecoder, debugMetadata, args)
{
	return _Platform_initialize(
		flagDecoder,
		args,
		impl.init,
		impl.update,
		impl.subscriptions,
		function() { return function() {} }
	);
});



// INITIALIZE A PROGRAM


function _Platform_initialize(flagDecoder, args, init, update, subscriptions, stepperBuilder)
{
	var result = A2(_Json_run, flagDecoder, _Json_wrap(args ? args['flags'] : undefined));
	$elm$core$Result$isOk(result) || _Debug_crash(2 /**/, _Json_errorToString(result.a) /**/);
	var managers = {};
	var initPair = init(result.a);
	var model = initPair.a;
	var stepper = stepperBuilder(sendToApp, model);
	var ports = _Platform_setupEffects(managers, sendToApp);

	function sendToApp(msg, viewMetadata)
	{
		var pair = A2(update, msg, model);
		stepper(model = pair.a, viewMetadata);
		_Platform_enqueueEffects(managers, pair.b, subscriptions(model));
	}

	_Platform_enqueueEffects(managers, initPair.b, subscriptions(model));

	return ports ? { ports: ports } : {};
}



// TRACK PRELOADS
//
// This is used by code in elm/browser and elm/http
// to register any HTTP requests that are triggered by init.
//


var _Platform_preload;


function _Platform_registerPreload(url)
{
	_Platform_preload.add(url);
}



// EFFECT MANAGERS


var _Platform_effectManagers = {};


function _Platform_setupEffects(managers, sendToApp)
{
	var ports;

	// setup all necessary effect managers
	for (var key in _Platform_effectManagers)
	{
		var manager = _Platform_effectManagers[key];

		if (manager.a)
		{
			ports = ports || {};
			ports[key] = manager.a(key, sendToApp);
		}

		managers[key] = _Platform_instantiateManager(manager, sendToApp);
	}

	return ports;
}


function _Platform_createManager(init, onEffects, onSelfMsg, cmdMap, subMap)
{
	return {
		b: init,
		c: onEffects,
		d: onSelfMsg,
		e: cmdMap,
		f: subMap
	};
}


function _Platform_instantiateManager(info, sendToApp)
{
	var router = {
		g: sendToApp,
		h: undefined
	};

	var onEffects = info.c;
	var onSelfMsg = info.d;
	var cmdMap = info.e;
	var subMap = info.f;

	function loop(state)
	{
		return A2(_Scheduler_andThen, loop, _Scheduler_receive(function(msg)
		{
			var value = msg.a;

			if (msg.$ === 0)
			{
				return A3(onSelfMsg, router, value, state);
			}

			return cmdMap && subMap
				? A4(onEffects, router, value.i, value.j, state)
				: A3(onEffects, router, cmdMap ? value.i : value.j, state);
		}));
	}

	return router.h = _Scheduler_rawSpawn(A2(_Scheduler_andThen, loop, info.b));
}



// ROUTING


var _Platform_sendToApp = F2(function(router, msg)
{
	return _Scheduler_binding(function(callback)
	{
		router.g(msg);
		callback(_Scheduler_succeed(_Utils_Tuple0));
	});
});


var _Platform_sendToSelf = F2(function(router, msg)
{
	return A2(_Scheduler_send, router.h, {
		$: 0,
		a: msg
	});
});



// BAGS


function _Platform_leaf(home)
{
	return function(value)
	{
		return {
			$: 1,
			k: home,
			l: value
		};
	};
}


function _Platform_batch(list)
{
	return {
		$: 2,
		m: list
	};
}


var _Platform_map = F2(function(tagger, bag)
{
	return {
		$: 3,
		n: tagger,
		o: bag
	}
});



// PIPE BAGS INTO EFFECT MANAGERS
//
// Effects must be queued!
//
// Say your init contains a synchronous command, like Time.now or Time.here
//
//   - This will produce a batch of effects (FX_1)
//   - The synchronous task triggers the subsequent `update` call
//   - This will produce a batch of effects (FX_2)
//
// If we just start dispatching FX_2, subscriptions from FX_2 can be processed
// before subscriptions from FX_1. No good! Earlier versions of this code had
// this problem, leading to these reports:
//
//   https://github.com/elm/core/issues/980
//   https://github.com/elm/core/pull/981
//   https://github.com/elm/compiler/issues/1776
//
// The queue is necessary to avoid ordering issues for synchronous commands.


// Why use true/false here? Why not just check the length of the queue?
// The goal is to detect "are we currently dispatching effects?" If we
// are, we need to bail and let the ongoing while loop handle things.
//
// Now say the queue has 1 element. When we dequeue the final element,
// the queue will be empty, but we are still actively dispatching effects.
// So you could get queue jumping in a really tricky category of cases.
//
var _Platform_effectsQueue = [];
var _Platform_effectsActive = false;


function _Platform_enqueueEffects(managers, cmdBag, subBag)
{
	_Platform_effectsQueue.push({ p: managers, q: cmdBag, r: subBag });

	if (_Platform_effectsActive) return;

	_Platform_effectsActive = true;
	for (var fx; fx = _Platform_effectsQueue.shift(); )
	{
		_Platform_dispatchEffects(fx.p, fx.q, fx.r);
	}
	_Platform_effectsActive = false;
}


function _Platform_dispatchEffects(managers, cmdBag, subBag)
{
	var effectsDict = {};
	_Platform_gatherEffects(true, cmdBag, effectsDict, null);
	_Platform_gatherEffects(false, subBag, effectsDict, null);

	for (var home in managers)
	{
		_Scheduler_rawSend(managers[home], {
			$: 'fx',
			a: effectsDict[home] || { i: _List_Nil, j: _List_Nil }
		});
	}
}


function _Platform_gatherEffects(isCmd, bag, effectsDict, taggers)
{
	switch (bag.$)
	{
		case 1:
			var home = bag.k;
			var effect = _Platform_toEffect(isCmd, home, taggers, bag.l);
			effectsDict[home] = _Platform_insert(isCmd, effect, effectsDict[home]);
			return;

		case 2:
			for (var list = bag.m; list.b; list = list.b) // WHILE_CONS
			{
				_Platform_gatherEffects(isCmd, list.a, effectsDict, taggers);
			}
			return;

		case 3:
			_Platform_gatherEffects(isCmd, bag.o, effectsDict, {
				s: bag.n,
				t: taggers
			});
			return;
	}
}


function _Platform_toEffect(isCmd, home, taggers, value)
{
	function applyTaggers(x)
	{
		for (var temp = taggers; temp; temp = temp.t)
		{
			x = temp.s(x);
		}
		return x;
	}

	var map = isCmd
		? _Platform_effectManagers[home].e
		: _Platform_effectManagers[home].f;

	return A2(map, applyTaggers, value)
}


function _Platform_insert(isCmd, newEffect, effects)
{
	effects = effects || { i: _List_Nil, j: _List_Nil };

	isCmd
		? (effects.i = _List_Cons(newEffect, effects.i))
		: (effects.j = _List_Cons(newEffect, effects.j));

	return effects;
}



// PORTS


function _Platform_checkPortName(name)
{
	if (_Platform_effectManagers[name])
	{
		_Debug_crash(3, name)
	}
}



// OUTGOING PORTS


function _Platform_outgoingPort(name, converter)
{
	_Platform_checkPortName(name);
	_Platform_effectManagers[name] = {
		e: _Platform_outgoingPortMap,
		u: converter,
		a: _Platform_setupOutgoingPort
	};
	return _Platform_leaf(name);
}


var _Platform_outgoingPortMap = F2(function(tagger, value) { return value; });


function _Platform_setupOutgoingPort(name)
{
	var subs = [];
	var converter = _Platform_effectManagers[name].u;

	// CREATE MANAGER

	var init = _Process_sleep(0);

	_Platform_effectManagers[name].b = init;
	_Platform_effectManagers[name].c = F3(function(router, cmdList, state)
	{
		for ( ; cmdList.b; cmdList = cmdList.b) // WHILE_CONS
		{
			// grab a separate reference to subs in case unsubscribe is called
			var currentSubs = subs;
			var value = _Json_unwrap(converter(cmdList.a));
			for (var i = 0; i < currentSubs.length; i++)
			{
				currentSubs[i](value);
			}
		}
		return init;
	});

	// PUBLIC API

	function subscribe(callback)
	{
		subs.push(callback);
	}

	function unsubscribe(callback)
	{
		// copy subs into a new array in case unsubscribe is called within a
		// subscribed callback
		subs = subs.slice();
		var index = subs.indexOf(callback);
		if (index >= 0)
		{
			subs.splice(index, 1);
		}
	}

	return {
		subscribe: subscribe,
		unsubscribe: unsubscribe
	};
}



// INCOMING PORTS


function _Platform_incomingPort(name, converter)
{
	_Platform_checkPortName(name);
	_Platform_effectManagers[name] = {
		f: _Platform_incomingPortMap,
		u: converter,
		a: _Platform_setupIncomingPort
	};
	return _Platform_leaf(name);
}


var _Platform_incomingPortMap = F2(function(tagger, finalTagger)
{
	return function(value)
	{
		return tagger(finalTagger(value));
	};
});


function _Platform_setupIncomingPort(name, sendToApp)
{
	var subs = _List_Nil;
	var converter = _Platform_effectManagers[name].u;

	// CREATE MANAGER

	var init = _Scheduler_succeed(null);

	_Platform_effectManagers[name].b = init;
	_Platform_effectManagers[name].c = F3(function(router, subList, state)
	{
		subs = subList;
		return init;
	});

	// PUBLIC API

	function send(incomingValue)
	{
		var result = A2(_Json_run, converter, _Json_wrap(incomingValue));

		$elm$core$Result$isOk(result) || _Debug_crash(4, name, result.a);

		var value = result.a;
		for (var temp = subs; temp.b; temp = temp.b) // WHILE_CONS
		{
			sendToApp(temp.a(value));
		}
	}

	return { send: send };
}



// EXPORT ELM MODULES
//
// Have DEBUG and PROD versions so that we can (1) give nicer errors in
// debug mode and (2) not pay for the bits needed for that in prod mode.
//


function _Platform_export_UNUSED(exports)
{
	scope['Elm']
		? _Platform_mergeExportsProd(scope['Elm'], exports)
		: scope['Elm'] = exports;
}


function _Platform_mergeExportsProd(obj, exports)
{
	for (var name in exports)
	{
		(name in obj)
			? (name == 'init')
				? _Debug_crash(6)
				: _Platform_mergeExportsProd(obj[name], exports[name])
			: (obj[name] = exports[name]);
	}
}


function _Platform_export(exports)
{
	scope['Elm']
		? _Platform_mergeExportsDebug('Elm', scope['Elm'], exports)
		: scope['Elm'] = exports;
}


function _Platform_mergeExportsDebug(moduleName, obj, exports)
{
	for (var name in exports)
	{
		(name in obj)
			? (name == 'init')
				? _Debug_crash(6, moduleName)
				: _Platform_mergeExportsDebug(moduleName + '.' + name, obj[name], exports[name])
			: (obj[name] = exports[name]);
	}
}




// HELPERS


var _VirtualDom_divertHrefToApp;

var _VirtualDom_doc = typeof document !== 'undefined' ? document : {};


function _VirtualDom_appendChild(parent, child)
{
	parent.appendChild(child);
}

var _VirtualDom_init = F4(function(virtualNode, flagDecoder, debugMetadata, args)
{
	// NOTE: this function needs _Platform_export available to work

	/**_UNUSED/
	var node = args['node'];
	//*/
	/**/
	var node = args && args['node'] ? args['node'] : _Debug_crash(0);
	//*/

	node.parentNode.replaceChild(
		_VirtualDom_render(virtualNode, function() {}),
		node
	);

	return {};
});



// TEXT


function _VirtualDom_text(string)
{
	return {
		$: 0,
		a: string
	};
}



// NODE


var _VirtualDom_nodeNS = F2(function(namespace, tag)
{
	return F2(function(factList, kidList)
	{
		for (var kids = [], descendantsCount = 0; kidList.b; kidList = kidList.b) // WHILE_CONS
		{
			var kid = kidList.a;
			descendantsCount += (kid.b || 0);
			kids.push(kid);
		}
		descendantsCount += kids.length;

		return {
			$: 1,
			c: tag,
			d: _VirtualDom_organizeFacts(factList),
			e: kids,
			f: namespace,
			b: descendantsCount
		};
	});
});


var _VirtualDom_node = _VirtualDom_nodeNS(undefined);



// KEYED NODE


var _VirtualDom_keyedNodeNS = F2(function(namespace, tag)
{
	return F2(function(factList, kidList)
	{
		for (var kids = [], descendantsCount = 0; kidList.b; kidList = kidList.b) // WHILE_CONS
		{
			var kid = kidList.a;
			descendantsCount += (kid.b.b || 0);
			kids.push(kid);
		}
		descendantsCount += kids.length;

		return {
			$: 2,
			c: tag,
			d: _VirtualDom_organizeFacts(factList),
			e: kids,
			f: namespace,
			b: descendantsCount
		};
	});
});


var _VirtualDom_keyedNode = _VirtualDom_keyedNodeNS(undefined);



// CUSTOM


function _VirtualDom_custom(factList, model, render, diff)
{
	return {
		$: 3,
		d: _VirtualDom_organizeFacts(factList),
		g: model,
		h: render,
		i: diff
	};
}



// MAP


var _VirtualDom_map = F2(function(tagger, node)
{
	return {
		$: 4,
		j: tagger,
		k: node,
		b: 1 + (node.b || 0)
	};
});



// LAZY


function _VirtualDom_thunk(refs, thunk)
{
	return {
		$: 5,
		l: refs,
		m: thunk,
		k: undefined
	};
}

var _VirtualDom_lazy = F2(function(func, a)
{
	return _VirtualDom_thunk([func, a], function() {
		return func(a);
	});
});

var _VirtualDom_lazy2 = F3(function(func, a, b)
{
	return _VirtualDom_thunk([func, a, b], function() {
		return A2(func, a, b);
	});
});

var _VirtualDom_lazy3 = F4(function(func, a, b, c)
{
	return _VirtualDom_thunk([func, a, b, c], function() {
		return A3(func, a, b, c);
	});
});

var _VirtualDom_lazy4 = F5(function(func, a, b, c, d)
{
	return _VirtualDom_thunk([func, a, b, c, d], function() {
		return A4(func, a, b, c, d);
	});
});

var _VirtualDom_lazy5 = F6(function(func, a, b, c, d, e)
{
	return _VirtualDom_thunk([func, a, b, c, d, e], function() {
		return A5(func, a, b, c, d, e);
	});
});

var _VirtualDom_lazy6 = F7(function(func, a, b, c, d, e, f)
{
	return _VirtualDom_thunk([func, a, b, c, d, e, f], function() {
		return A6(func, a, b, c, d, e, f);
	});
});

var _VirtualDom_lazy7 = F8(function(func, a, b, c, d, e, f, g)
{
	return _VirtualDom_thunk([func, a, b, c, d, e, f, g], function() {
		return A7(func, a, b, c, d, e, f, g);
	});
});

var _VirtualDom_lazy8 = F9(function(func, a, b, c, d, e, f, g, h)
{
	return _VirtualDom_thunk([func, a, b, c, d, e, f, g, h], function() {
		return A8(func, a, b, c, d, e, f, g, h);
	});
});



// FACTS


var _VirtualDom_on = F2(function(key, handler)
{
	return {
		$: 'a0',
		n: key,
		o: handler
	};
});
var _VirtualDom_style = F2(function(key, value)
{
	return {
		$: 'a1',
		n: key,
		o: value
	};
});
var _VirtualDom_property = F2(function(key, value)
{
	return {
		$: 'a2',
		n: key,
		o: value
	};
});
var _VirtualDom_attribute = F2(function(key, value)
{
	return {
		$: 'a3',
		n: key,
		o: value
	};
});
var _VirtualDom_attributeNS = F3(function(namespace, key, value)
{
	return {
		$: 'a4',
		n: key,
		o: { f: namespace, o: value }
	};
});



// XSS ATTACK VECTOR CHECKS
//
// For some reason, tabs can appear in href protocols and it still works.
// So '\tjava\tSCRIPT:alert("!!!")' and 'javascript:alert("!!!")' are the same
// in practice. That is why _VirtualDom_RE_js and _VirtualDom_RE_js_html look
// so freaky.
//
// Pulling the regular expressions out to the top level gives a slight speed
// boost in small benchmarks (4-10%) but hoisting values to reduce allocation
// can be unpredictable in large programs where JIT may have a harder time with
// functions are not fully self-contained. The benefit is more that the js and
// js_html ones are so weird that I prefer to see them near each other.


var _VirtualDom_RE_script = /^script$/i;
var _VirtualDom_RE_on_formAction = /^(on|formAction$)/i;
var _VirtualDom_RE_js = /^\s*j\s*a\s*v\s*a\s*s\s*c\s*r\s*i\s*p\s*t\s*:/i;
var _VirtualDom_RE_js_html = /^\s*(j\s*a\s*v\s*a\s*s\s*c\s*r\s*i\s*p\s*t\s*:|d\s*a\s*t\s*a\s*:\s*t\s*e\s*x\s*t\s*\/\s*h\s*t\s*m\s*l\s*(,|;))/i;


function _VirtualDom_noScript(tag)
{
	return _VirtualDom_RE_script.test(tag) ? 'p' : tag;
}

function _VirtualDom_noOnOrFormAction(key)
{
	return _VirtualDom_RE_on_formAction.test(key) ? 'data-' + key : key;
}

function _VirtualDom_noInnerHtmlOrFormAction(key)
{
	return key == 'innerHTML' || key == 'formAction' ? 'data-' + key : key;
}

function _VirtualDom_noJavaScriptUri(value)
{
	return _VirtualDom_RE_js.test(value)
		? /**_UNUSED/''//*//**/'javascript:alert("This is an XSS vector. Please use ports or web components instead.")'//*/
		: value;
}

function _VirtualDom_noJavaScriptOrHtmlUri(value)
{
	return _VirtualDom_RE_js_html.test(value)
		? /**_UNUSED/''//*//**/'javascript:alert("This is an XSS vector. Please use ports or web components instead.")'//*/
		: value;
}

function _VirtualDom_noJavaScriptOrHtmlJson(value)
{
	return (typeof _Json_unwrap(value) === 'string' && _VirtualDom_RE_js_html.test(_Json_unwrap(value)))
		? _Json_wrap(
			/**_UNUSED/''//*//**/'javascript:alert("This is an XSS vector. Please use ports or web components instead.")'//*/
		) : value;
}



// MAP FACTS


var _VirtualDom_mapAttribute = F2(function(func, attr)
{
	return (attr.$ === 'a0')
		? A2(_VirtualDom_on, attr.n, _VirtualDom_mapHandler(func, attr.o))
		: attr;
});

function _VirtualDom_mapHandler(func, handler)
{
	var tag = $elm$virtual_dom$VirtualDom$toHandlerInt(handler);

	// 0 = Normal
	// 1 = MayStopPropagation
	// 2 = MayPreventDefault
	// 3 = Custom

	return {
		$: handler.$,
		a:
			!tag
				? A2($elm$json$Json$Decode$map, func, handler.a)
				:
			A3($elm$json$Json$Decode$map2,
				tag < 3
					? _VirtualDom_mapEventTuple
					: _VirtualDom_mapEventRecord,
				$elm$json$Json$Decode$succeed(func),
				handler.a
			)
	};
}

var _VirtualDom_mapEventTuple = F2(function(func, tuple)
{
	return _Utils_Tuple2(func(tuple.a), tuple.b);
});

var _VirtualDom_mapEventRecord = F2(function(func, record)
{
	return {
		message: func(record.message),
		stopPropagation: record.stopPropagation,
		preventDefault: record.preventDefault
	}
});



// ORGANIZE FACTS


function _VirtualDom_organizeFacts(factList)
{
	for (var facts = {}; factList.b; factList = factList.b) // WHILE_CONS
	{
		var entry = factList.a;

		var tag = entry.$;
		var key = entry.n;
		var value = entry.o;

		if (tag === 'a2')
		{
			(key === 'className')
				? _VirtualDom_addClass(facts, key, _Json_unwrap(value))
				: facts[key] = _Json_unwrap(value);

			continue;
		}

		var subFacts = facts[tag] || (facts[tag] = {});
		(tag === 'a3' && key === 'class')
			? _VirtualDom_addClass(subFacts, key, value)
			: subFacts[key] = value;
	}

	return facts;
}

function _VirtualDom_addClass(object, key, newClass)
{
	var classes = object[key];
	object[key] = classes ? classes + ' ' + newClass : newClass;
}



// RENDER


function _VirtualDom_render(vNode, eventNode)
{
	var tag = vNode.$;

	if (tag === 5)
	{
		return _VirtualDom_render(vNode.k || (vNode.k = vNode.m()), eventNode);
	}

	if (tag === 0)
	{
		return _VirtualDom_doc.createTextNode(vNode.a);
	}

	if (tag === 4)
	{
		var subNode = vNode.k;
		var tagger = vNode.j;

		while (subNode.$ === 4)
		{
			typeof tagger !== 'object'
				? tagger = [tagger, subNode.j]
				: tagger.push(subNode.j);

			subNode = subNode.k;
		}

		var subEventRoot = { j: tagger, p: eventNode };
		var domNode = _VirtualDom_render(subNode, subEventRoot);
		domNode.elm_event_node_ref = subEventRoot;
		return domNode;
	}

	if (tag === 3)
	{
		var domNode = vNode.h(vNode.g);
		_VirtualDom_applyFacts(domNode, eventNode, vNode.d);
		return domNode;
	}

	// at this point `tag` must be 1 or 2

	var domNode = vNode.f
		? _VirtualDom_doc.createElementNS(vNode.f, vNode.c)
		: _VirtualDom_doc.createElement(vNode.c);

	if (_VirtualDom_divertHrefToApp && vNode.c == 'a')
	{
		domNode.addEventListener('click', _VirtualDom_divertHrefToApp(domNode));
	}

	_VirtualDom_applyFacts(domNode, eventNode, vNode.d);

	for (var kids = vNode.e, i = 0; i < kids.length; i++)
	{
		_VirtualDom_appendChild(domNode, _VirtualDom_render(tag === 1 ? kids[i] : kids[i].b, eventNode));
	}

	return domNode;
}



// APPLY FACTS


function _VirtualDom_applyFacts(domNode, eventNode, facts)
{
	for (var key in facts)
	{
		var value = facts[key];

		key === 'a1'
			? _VirtualDom_applyStyles(domNode, value)
			:
		key === 'a0'
			? _VirtualDom_applyEvents(domNode, eventNode, value)
			:
		key === 'a3'
			? _VirtualDom_applyAttrs(domNode, value)
			:
		key === 'a4'
			? _VirtualDom_applyAttrsNS(domNode, value)
			:
		((key !== 'value' && key !== 'checked') || domNode[key] !== value) && (domNode[key] = value);
	}
}



// APPLY STYLES


function _VirtualDom_applyStyles(domNode, styles)
{
	var domNodeStyle = domNode.style;

	for (var key in styles)
	{
		domNodeStyle[key] = styles[key];
	}
}



// APPLY ATTRS


function _VirtualDom_applyAttrs(domNode, attrs)
{
	for (var key in attrs)
	{
		var value = attrs[key];
		typeof value !== 'undefined'
			? domNode.setAttribute(key, value)
			: domNode.removeAttribute(key);
	}
}



// APPLY NAMESPACED ATTRS


function _VirtualDom_applyAttrsNS(domNode, nsAttrs)
{
	for (var key in nsAttrs)
	{
		var pair = nsAttrs[key];
		var namespace = pair.f;
		var value = pair.o;

		typeof value !== 'undefined'
			? domNode.setAttributeNS(namespace, key, value)
			: domNode.removeAttributeNS(namespace, key);
	}
}



// APPLY EVENTS


function _VirtualDom_applyEvents(domNode, eventNode, events)
{
	var allCallbacks = domNode.elmFs || (domNode.elmFs = {});

	for (var key in events)
	{
		var newHandler = events[key];
		var oldCallback = allCallbacks[key];

		if (!newHandler)
		{
			domNode.removeEventListener(key, oldCallback);
			allCallbacks[key] = undefined;
			continue;
		}

		if (oldCallback)
		{
			var oldHandler = oldCallback.q;
			if (oldHandler.$ === newHandler.$)
			{
				oldCallback.q = newHandler;
				continue;
			}
			domNode.removeEventListener(key, oldCallback);
		}

		oldCallback = _VirtualDom_makeCallback(eventNode, newHandler);
		domNode.addEventListener(key, oldCallback,
			_VirtualDom_passiveSupported
			&& { passive: $elm$virtual_dom$VirtualDom$toHandlerInt(newHandler) < 2 }
		);
		allCallbacks[key] = oldCallback;
	}
}



// PASSIVE EVENTS


var _VirtualDom_passiveSupported;

try
{
	window.addEventListener('t', null, Object.defineProperty({}, 'passive', {
		get: function() { _VirtualDom_passiveSupported = true; }
	}));
}
catch(e) {}



// EVENT HANDLERS


function _VirtualDom_makeCallback(eventNode, initialHandler)
{
	function callback(event)
	{
		var handler = callback.q;
		var result = _Json_runHelp(handler.a, event);

		if (!$elm$core$Result$isOk(result))
		{
			return;
		}

		var tag = $elm$virtual_dom$VirtualDom$toHandlerInt(handler);

		// 0 = Normal
		// 1 = MayStopPropagation
		// 2 = MayPreventDefault
		// 3 = Custom

		var value = result.a;
		var message = !tag ? value : tag < 3 ? value.a : value.message;
		var stopPropagation = tag == 1 ? value.b : tag == 3 && value.stopPropagation;
		var currentEventNode = (
			stopPropagation && event.stopPropagation(),
			(tag == 2 ? value.b : tag == 3 && value.preventDefault) && event.preventDefault(),
			eventNode
		);
		var tagger;
		var i;
		while (tagger = currentEventNode.j)
		{
			if (typeof tagger == 'function')
			{
				message = tagger(message);
			}
			else
			{
				for (var i = tagger.length; i--; )
				{
					message = tagger[i](message);
				}
			}
			currentEventNode = currentEventNode.p;
		}
		currentEventNode(message, stopPropagation); // stopPropagation implies isSync
	}

	callback.q = initialHandler;

	return callback;
}

function _VirtualDom_equalEvents(x, y)
{
	return x.$ == y.$ && _Json_equality(x.a, y.a);
}



// DIFF


// TODO: Should we do patches like in iOS?
//
// type Patch
//   = At Int Patch
//   | Batch (List Patch)
//   | Change ...
//
// How could it not be better?
//
function _VirtualDom_diff(x, y)
{
	var patches = [];
	_VirtualDom_diffHelp(x, y, patches, 0);
	return patches;
}


function _VirtualDom_pushPatch(patches, type, index, data)
{
	var patch = {
		$: type,
		r: index,
		s: data,
		t: undefined,
		u: undefined
	};
	patches.push(patch);
	return patch;
}


function _VirtualDom_diffHelp(x, y, patches, index)
{
	if (x === y)
	{
		return;
	}

	var xType = x.$;
	var yType = y.$;

	// Bail if you run into different types of nodes. Implies that the
	// structure has changed significantly and it's not worth a diff.
	if (xType !== yType)
	{
		if (xType === 1 && yType === 2)
		{
			y = _VirtualDom_dekey(y);
			yType = 1;
		}
		else
		{
			_VirtualDom_pushPatch(patches, 0, index, y);
			return;
		}
	}

	// Now we know that both nodes are the same $.
	switch (yType)
	{
		case 5:
			var xRefs = x.l;
			var yRefs = y.l;
			var i = xRefs.length;
			var same = i === yRefs.length;
			while (same && i--)
			{
				same = xRefs[i] === yRefs[i];
			}
			if (same)
			{
				y.k = x.k;
				return;
			}
			y.k = y.m();
			var subPatches = [];
			_VirtualDom_diffHelp(x.k, y.k, subPatches, 0);
			subPatches.length > 0 && _VirtualDom_pushPatch(patches, 1, index, subPatches);
			return;

		case 4:
			// gather nested taggers
			var xTaggers = x.j;
			var yTaggers = y.j;
			var nesting = false;

			var xSubNode = x.k;
			while (xSubNode.$ === 4)
			{
				nesting = true;

				typeof xTaggers !== 'object'
					? xTaggers = [xTaggers, xSubNode.j]
					: xTaggers.push(xSubNode.j);

				xSubNode = xSubNode.k;
			}

			var ySubNode = y.k;
			while (ySubNode.$ === 4)
			{
				nesting = true;

				typeof yTaggers !== 'object'
					? yTaggers = [yTaggers, ySubNode.j]
					: yTaggers.push(ySubNode.j);

				ySubNode = ySubNode.k;
			}

			// Just bail if different numbers of taggers. This implies the
			// structure of the virtual DOM has changed.
			if (nesting && xTaggers.length !== yTaggers.length)
			{
				_VirtualDom_pushPatch(patches, 0, index, y);
				return;
			}

			// check if taggers are "the same"
			if (nesting ? !_VirtualDom_pairwiseRefEqual(xTaggers, yTaggers) : xTaggers !== yTaggers)
			{
				_VirtualDom_pushPatch(patches, 2, index, yTaggers);
			}

			// diff everything below the taggers
			_VirtualDom_diffHelp(xSubNode, ySubNode, patches, index + 1);
			return;

		case 0:
			if (x.a !== y.a)
			{
				_VirtualDom_pushPatch(patches, 3, index, y.a);
			}
			return;

		case 1:
			_VirtualDom_diffNodes(x, y, patches, index, _VirtualDom_diffKids);
			return;

		case 2:
			_VirtualDom_diffNodes(x, y, patches, index, _VirtualDom_diffKeyedKids);
			return;

		case 3:
			if (x.h !== y.h)
			{
				_VirtualDom_pushPatch(patches, 0, index, y);
				return;
			}

			var factsDiff = _VirtualDom_diffFacts(x.d, y.d);
			factsDiff && _VirtualDom_pushPatch(patches, 4, index, factsDiff);

			var patch = y.i(x.g, y.g);
			patch && _VirtualDom_pushPatch(patches, 5, index, patch);

			return;
	}
}

// assumes the incoming arrays are the same length
function _VirtualDom_pairwiseRefEqual(as, bs)
{
	for (var i = 0; i < as.length; i++)
	{
		if (as[i] !== bs[i])
		{
			return false;
		}
	}

	return true;
}

function _VirtualDom_diffNodes(x, y, patches, index, diffKids)
{
	// Bail if obvious indicators have changed. Implies more serious
	// structural changes such that it's not worth it to diff.
	if (x.c !== y.c || x.f !== y.f)
	{
		_VirtualDom_pushPatch(patches, 0, index, y);
		return;
	}

	var factsDiff = _VirtualDom_diffFacts(x.d, y.d);
	factsDiff && _VirtualDom_pushPatch(patches, 4, index, factsDiff);

	diffKids(x, y, patches, index);
}



// DIFF FACTS


// TODO Instead of creating a new diff object, it's possible to just test if
// there *is* a diff. During the actual patch, do the diff again and make the
// modifications directly. This way, there's no new allocations. Worth it?
function _VirtualDom_diffFacts(x, y, category)
{
	var diff;

	// look for changes and removals
	for (var xKey in x)
	{
		if (xKey === 'a1' || xKey === 'a0' || xKey === 'a3' || xKey === 'a4')
		{
			var subDiff = _VirtualDom_diffFacts(x[xKey], y[xKey] || {}, xKey);
			if (subDiff)
			{
				diff = diff || {};
				diff[xKey] = subDiff;
			}
			continue;
		}

		// remove if not in the new facts
		if (!(xKey in y))
		{
			diff = diff || {};
			diff[xKey] =
				!category
					? (typeof x[xKey] === 'string' ? '' : null)
					:
				(category === 'a1')
					? ''
					:
				(category === 'a0' || category === 'a3')
					? undefined
					:
				{ f: x[xKey].f, o: undefined };

			continue;
		}

		var xValue = x[xKey];
		var yValue = y[xKey];

		// reference equal, so don't worry about it
		if (xValue === yValue && xKey !== 'value' && xKey !== 'checked'
			|| category === 'a0' && _VirtualDom_equalEvents(xValue, yValue))
		{
			continue;
		}

		diff = diff || {};
		diff[xKey] = yValue;
	}

	// add new stuff
	for (var yKey in y)
	{
		if (!(yKey in x))
		{
			diff = diff || {};
			diff[yKey] = y[yKey];
		}
	}

	return diff;
}



// DIFF KIDS


function _VirtualDom_diffKids(xParent, yParent, patches, index)
{
	var xKids = xParent.e;
	var yKids = yParent.e;

	var xLen = xKids.length;
	var yLen = yKids.length;

	// FIGURE OUT IF THERE ARE INSERTS OR REMOVALS

	if (xLen > yLen)
	{
		_VirtualDom_pushPatch(patches, 6, index, {
			v: yLen,
			i: xLen - yLen
		});
	}
	else if (xLen < yLen)
	{
		_VirtualDom_pushPatch(patches, 7, index, {
			v: xLen,
			e: yKids
		});
	}

	// PAIRWISE DIFF EVERYTHING ELSE

	for (var minLen = xLen < yLen ? xLen : yLen, i = 0; i < minLen; i++)
	{
		var xKid = xKids[i];
		_VirtualDom_diffHelp(xKid, yKids[i], patches, ++index);
		index += xKid.b || 0;
	}
}



// KEYED DIFF


function _VirtualDom_diffKeyedKids(xParent, yParent, patches, rootIndex)
{
	var localPatches = [];

	var changes = {}; // Dict String Entry
	var inserts = []; // Array { index : Int, entry : Entry }
	// type Entry = { tag : String, vnode : VNode, index : Int, data : _ }

	var xKids = xParent.e;
	var yKids = yParent.e;
	var xLen = xKids.length;
	var yLen = yKids.length;
	var xIndex = 0;
	var yIndex = 0;

	var index = rootIndex;

	while (xIndex < xLen && yIndex < yLen)
	{
		var x = xKids[xIndex];
		var y = yKids[yIndex];

		var xKey = x.a;
		var yKey = y.a;
		var xNode = x.b;
		var yNode = y.b;

		var newMatch = undefined;
		var oldMatch = undefined;

		// check if keys match

		if (xKey === yKey)
		{
			index++;
			_VirtualDom_diffHelp(xNode, yNode, localPatches, index);
			index += xNode.b || 0;

			xIndex++;
			yIndex++;
			continue;
		}

		// look ahead 1 to detect insertions and removals.

		var xNext = xKids[xIndex + 1];
		var yNext = yKids[yIndex + 1];

		if (xNext)
		{
			var xNextKey = xNext.a;
			var xNextNode = xNext.b;
			oldMatch = yKey === xNextKey;
		}

		if (yNext)
		{
			var yNextKey = yNext.a;
			var yNextNode = yNext.b;
			newMatch = xKey === yNextKey;
		}


		// swap x and y
		if (newMatch && oldMatch)
		{
			index++;
			_VirtualDom_diffHelp(xNode, yNextNode, localPatches, index);
			_VirtualDom_insertNode(changes, localPatches, xKey, yNode, yIndex, inserts);
			index += xNode.b || 0;

			index++;
			_VirtualDom_removeNode(changes, localPatches, xKey, xNextNode, index);
			index += xNextNode.b || 0;

			xIndex += 2;
			yIndex += 2;
			continue;
		}

		// insert y
		if (newMatch)
		{
			index++;
			_VirtualDom_insertNode(changes, localPatches, yKey, yNode, yIndex, inserts);
			_VirtualDom_diffHelp(xNode, yNextNode, localPatches, index);
			index += xNode.b || 0;

			xIndex += 1;
			yIndex += 2;
			continue;
		}

		// remove x
		if (oldMatch)
		{
			index++;
			_VirtualDom_removeNode(changes, localPatches, xKey, xNode, index);
			index += xNode.b || 0;

			index++;
			_VirtualDom_diffHelp(xNextNode, yNode, localPatches, index);
			index += xNextNode.b || 0;

			xIndex += 2;
			yIndex += 1;
			continue;
		}

		// remove x, insert y
		if (xNext && xNextKey === yNextKey)
		{
			index++;
			_VirtualDom_removeNode(changes, localPatches, xKey, xNode, index);
			_VirtualDom_insertNode(changes, localPatches, yKey, yNode, yIndex, inserts);
			index += xNode.b || 0;

			index++;
			_VirtualDom_diffHelp(xNextNode, yNextNode, localPatches, index);
			index += xNextNode.b || 0;

			xIndex += 2;
			yIndex += 2;
			continue;
		}

		break;
	}

	// eat up any remaining nodes with removeNode and insertNode

	while (xIndex < xLen)
	{
		index++;
		var x = xKids[xIndex];
		var xNode = x.b;
		_VirtualDom_removeNode(changes, localPatches, x.a, xNode, index);
		index += xNode.b || 0;
		xIndex++;
	}

	while (yIndex < yLen)
	{
		var endInserts = endInserts || [];
		var y = yKids[yIndex];
		_VirtualDom_insertNode(changes, localPatches, y.a, y.b, undefined, endInserts);
		yIndex++;
	}

	if (localPatches.length > 0 || inserts.length > 0 || endInserts)
	{
		_VirtualDom_pushPatch(patches, 8, rootIndex, {
			w: localPatches,
			x: inserts,
			y: endInserts
		});
	}
}



// CHANGES FROM KEYED DIFF


var _VirtualDom_POSTFIX = '_elmW6BL';


function _VirtualDom_insertNode(changes, localPatches, key, vnode, yIndex, inserts)
{
	var entry = changes[key];

	// never seen this key before
	if (!entry)
	{
		entry = {
			c: 0,
			z: vnode,
			r: yIndex,
			s: undefined
		};

		inserts.push({ r: yIndex, A: entry });
		changes[key] = entry;

		return;
	}

	// this key was removed earlier, a match!
	if (entry.c === 1)
	{
		inserts.push({ r: yIndex, A: entry });

		entry.c = 2;
		var subPatches = [];
		_VirtualDom_diffHelp(entry.z, vnode, subPatches, entry.r);
		entry.r = yIndex;
		entry.s.s = {
			w: subPatches,
			A: entry
		};

		return;
	}

	// this key has already been inserted or moved, a duplicate!
	_VirtualDom_insertNode(changes, localPatches, key + _VirtualDom_POSTFIX, vnode, yIndex, inserts);
}


function _VirtualDom_removeNode(changes, localPatches, key, vnode, index)
{
	var entry = changes[key];

	// never seen this key before
	if (!entry)
	{
		var patch = _VirtualDom_pushPatch(localPatches, 9, index, undefined);

		changes[key] = {
			c: 1,
			z: vnode,
			r: index,
			s: patch
		};

		return;
	}

	// this key was inserted earlier, a match!
	if (entry.c === 0)
	{
		entry.c = 2;
		var subPatches = [];
		_VirtualDom_diffHelp(vnode, entry.z, subPatches, index);

		_VirtualDom_pushPatch(localPatches, 9, index, {
			w: subPatches,
			A: entry
		});

		return;
	}

	// this key has already been removed or moved, a duplicate!
	_VirtualDom_removeNode(changes, localPatches, key + _VirtualDom_POSTFIX, vnode, index);
}



// ADD DOM NODES
//
// Each DOM node has an "index" assigned in order of traversal. It is important
// to minimize our crawl over the actual DOM, so these indexes (along with the
// descendantsCount of virtual nodes) let us skip touching entire subtrees of
// the DOM if we know there are no patches there.


function _VirtualDom_addDomNodes(domNode, vNode, patches, eventNode)
{
	_VirtualDom_addDomNodesHelp(domNode, vNode, patches, 0, 0, vNode.b, eventNode);
}


// assumes `patches` is non-empty and indexes increase monotonically.
function _VirtualDom_addDomNodesHelp(domNode, vNode, patches, i, low, high, eventNode)
{
	var patch = patches[i];
	var index = patch.r;

	while (index === low)
	{
		var patchType = patch.$;

		if (patchType === 1)
		{
			_VirtualDom_addDomNodes(domNode, vNode.k, patch.s, eventNode);
		}
		else if (patchType === 8)
		{
			patch.t = domNode;
			patch.u = eventNode;

			var subPatches = patch.s.w;
			if (subPatches.length > 0)
			{
				_VirtualDom_addDomNodesHelp(domNode, vNode, subPatches, 0, low, high, eventNode);
			}
		}
		else if (patchType === 9)
		{
			patch.t = domNode;
			patch.u = eventNode;

			var data = patch.s;
			if (data)
			{
				data.A.s = domNode;
				var subPatches = data.w;
				if (subPatches.length > 0)
				{
					_VirtualDom_addDomNodesHelp(domNode, vNode, subPatches, 0, low, high, eventNode);
				}
			}
		}
		else
		{
			patch.t = domNode;
			patch.u = eventNode;
		}

		i++;

		if (!(patch = patches[i]) || (index = patch.r) > high)
		{
			return i;
		}
	}

	var tag = vNode.$;

	if (tag === 4)
	{
		var subNode = vNode.k;

		while (subNode.$ === 4)
		{
			subNode = subNode.k;
		}

		return _VirtualDom_addDomNodesHelp(domNode, subNode, patches, i, low + 1, high, domNode.elm_event_node_ref);
	}

	// tag must be 1 or 2 at this point

	var vKids = vNode.e;
	var childNodes = domNode.childNodes;
	for (var j = 0; j < vKids.length; j++)
	{
		low++;
		var vKid = tag === 1 ? vKids[j] : vKids[j].b;
		var nextLow = low + (vKid.b || 0);
		if (low <= index && index <= nextLow)
		{
			i = _VirtualDom_addDomNodesHelp(childNodes[j], vKid, patches, i, low, nextLow, eventNode);
			if (!(patch = patches[i]) || (index = patch.r) > high)
			{
				return i;
			}
		}
		low = nextLow;
	}
	return i;
}



// APPLY PATCHES


function _VirtualDom_applyPatches(rootDomNode, oldVirtualNode, patches, eventNode)
{
	if (patches.length === 0)
	{
		return rootDomNode;
	}

	_VirtualDom_addDomNodes(rootDomNode, oldVirtualNode, patches, eventNode);
	return _VirtualDom_applyPatchesHelp(rootDomNode, patches);
}

function _VirtualDom_applyPatchesHelp(rootDomNode, patches)
{
	for (var i = 0; i < patches.length; i++)
	{
		var patch = patches[i];
		var localDomNode = patch.t
		var newNode = _VirtualDom_applyPatch(localDomNode, patch);
		if (localDomNode === rootDomNode)
		{
			rootDomNode = newNode;
		}
	}
	return rootDomNode;
}

function _VirtualDom_applyPatch(domNode, patch)
{
	switch (patch.$)
	{
		case 0:
			return _VirtualDom_applyPatchRedraw(domNode, patch.s, patch.u);

		case 4:
			_VirtualDom_applyFacts(domNode, patch.u, patch.s);
			return domNode;

		case 3:
			domNode.replaceData(0, domNode.length, patch.s);
			return domNode;

		case 1:
			return _VirtualDom_applyPatchesHelp(domNode, patch.s);

		case 2:
			if (domNode.elm_event_node_ref)
			{
				domNode.elm_event_node_ref.j = patch.s;
			}
			else
			{
				domNode.elm_event_node_ref = { j: patch.s, p: patch.u };
			}
			return domNode;

		case 6:
			var data = patch.s;
			for (var i = 0; i < data.i; i++)
			{
				domNode.removeChild(domNode.childNodes[data.v]);
			}
			return domNode;

		case 7:
			var data = patch.s;
			var kids = data.e;
			var i = data.v;
			var theEnd = domNode.childNodes[i];
			for (; i < kids.length; i++)
			{
				domNode.insertBefore(_VirtualDom_render(kids[i], patch.u), theEnd);
			}
			return domNode;

		case 9:
			var data = patch.s;
			if (!data)
			{
				domNode.parentNode.removeChild(domNode);
				return domNode;
			}
			var entry = data.A;
			if (typeof entry.r !== 'undefined')
			{
				domNode.parentNode.removeChild(domNode);
			}
			entry.s = _VirtualDom_applyPatchesHelp(domNode, data.w);
			return domNode;

		case 8:
			return _VirtualDom_applyPatchReorder(domNode, patch);

		case 5:
			return patch.s(domNode);

		default:
			_Debug_crash(10); // 'Ran into an unknown patch!'
	}
}


function _VirtualDom_applyPatchRedraw(domNode, vNode, eventNode)
{
	var parentNode = domNode.parentNode;
	var newNode = _VirtualDom_render(vNode, eventNode);

	if (!newNode.elm_event_node_ref)
	{
		newNode.elm_event_node_ref = domNode.elm_event_node_ref;
	}

	if (parentNode && newNode !== domNode)
	{
		parentNode.replaceChild(newNode, domNode);
	}
	return newNode;
}


function _VirtualDom_applyPatchReorder(domNode, patch)
{
	var data = patch.s;

	// remove end inserts
	var frag = _VirtualDom_applyPatchReorderEndInsertsHelp(data.y, patch);

	// removals
	domNode = _VirtualDom_applyPatchesHelp(domNode, data.w);

	// inserts
	var inserts = data.x;
	for (var i = 0; i < inserts.length; i++)
	{
		var insert = inserts[i];
		var entry = insert.A;
		var node = entry.c === 2
			? entry.s
			: _VirtualDom_render(entry.z, patch.u);
		domNode.insertBefore(node, domNode.childNodes[insert.r]);
	}

	// add end inserts
	if (frag)
	{
		_VirtualDom_appendChild(domNode, frag);
	}

	return domNode;
}


function _VirtualDom_applyPatchReorderEndInsertsHelp(endInserts, patch)
{
	if (!endInserts)
	{
		return;
	}

	var frag = _VirtualDom_doc.createDocumentFragment();
	for (var i = 0; i < endInserts.length; i++)
	{
		var insert = endInserts[i];
		var entry = insert.A;
		_VirtualDom_appendChild(frag, entry.c === 2
			? entry.s
			: _VirtualDom_render(entry.z, patch.u)
		);
	}
	return frag;
}


function _VirtualDom_virtualize(node)
{
	// TEXT NODES

	if (node.nodeType === 3)
	{
		return _VirtualDom_text(node.textContent);
	}


	// WEIRD NODES

	if (node.nodeType !== 1)
	{
		return _VirtualDom_text('');
	}


	// ELEMENT NODES

	var attrList = _List_Nil;
	var attrs = node.attributes;
	for (var i = attrs.length; i--; )
	{
		var attr = attrs[i];
		var name = attr.name;
		var value = attr.value;
		attrList = _List_Cons( A2(_VirtualDom_attribute, name, value), attrList );
	}

	var tag = node.tagName.toLowerCase();
	var kidList = _List_Nil;
	var kids = node.childNodes;

	for (var i = kids.length; i--; )
	{
		kidList = _List_Cons(_VirtualDom_virtualize(kids[i]), kidList);
	}
	return A3(_VirtualDom_node, tag, attrList, kidList);
}

function _VirtualDom_dekey(keyedNode)
{
	var keyedKids = keyedNode.e;
	var len = keyedKids.length;
	var kids = new Array(len);
	for (var i = 0; i < len; i++)
	{
		kids[i] = keyedKids[i].b;
	}

	return {
		$: 1,
		c: keyedNode.c,
		d: keyedNode.d,
		e: kids,
		f: keyedNode.f,
		b: keyedNode.b
	};
}




// ELEMENT


var _Debugger_element;

var _Browser_element = _Debugger_element || F4(function(impl, flagDecoder, debugMetadata, args)
{
	return _Platform_initialize(
		flagDecoder,
		args,
		impl.init,
		impl.update,
		impl.subscriptions,
		function(sendToApp, initialModel) {
			var view = impl.view;
			/**_UNUSED/
			var domNode = args['node'];
			//*/
			/**/
			var domNode = args && args['node'] ? args['node'] : _Debug_crash(0);
			//*/
			var currNode = _VirtualDom_virtualize(domNode);

			return _Browser_makeAnimator(initialModel, function(model)
			{
				var nextNode = view(model);
				var patches = _VirtualDom_diff(currNode, nextNode);
				domNode = _VirtualDom_applyPatches(domNode, currNode, patches, sendToApp);
				currNode = nextNode;
			});
		}
	);
});



// DOCUMENT


var _Debugger_document;

var _Browser_document = _Debugger_document || F4(function(impl, flagDecoder, debugMetadata, args)
{
	return _Platform_initialize(
		flagDecoder,
		args,
		impl.init,
		impl.update,
		impl.subscriptions,
		function(sendToApp, initialModel) {
			var divertHrefToApp = impl.setup && impl.setup(sendToApp)
			var view = impl.view;
			var title = _VirtualDom_doc.title;
			var bodyNode = _VirtualDom_doc.body;
			var currNode = _VirtualDom_virtualize(bodyNode);
			return _Browser_makeAnimator(initialModel, function(model)
			{
				_VirtualDom_divertHrefToApp = divertHrefToApp;
				var doc = view(model);
				var nextNode = _VirtualDom_node('body')(_List_Nil)(doc.body);
				var patches = _VirtualDom_diff(currNode, nextNode);
				bodyNode = _VirtualDom_applyPatches(bodyNode, currNode, patches, sendToApp);
				currNode = nextNode;
				_VirtualDom_divertHrefToApp = 0;
				(title !== doc.title) && (_VirtualDom_doc.title = title = doc.title);
			});
		}
	);
});



// ANIMATION


var _Browser_cancelAnimationFrame =
	typeof cancelAnimationFrame !== 'undefined'
		? cancelAnimationFrame
		: function(id) { clearTimeout(id); };

var _Browser_requestAnimationFrame =
	typeof requestAnimationFrame !== 'undefined'
		? requestAnimationFrame
		: function(callback) { return setTimeout(callback, 1000 / 60); };


function _Browser_makeAnimator(model, draw)
{
	draw(model);

	var state = 0;

	function updateIfNeeded()
	{
		state = state === 1
			? 0
			: ( _Browser_requestAnimationFrame(updateIfNeeded), draw(model), 1 );
	}

	return function(nextModel, isSync)
	{
		model = nextModel;

		isSync
			? ( draw(model),
				state === 2 && (state = 1)
				)
			: ( state === 0 && _Browser_requestAnimationFrame(updateIfNeeded),
				state = 2
				);
	};
}



// APPLICATION


function _Browser_application(impl)
{
	var onUrlChange = impl.onUrlChange;
	var onUrlRequest = impl.onUrlRequest;
	var key = function() { key.a(onUrlChange(_Browser_getUrl())); };

	return _Browser_document({
		setup: function(sendToApp)
		{
			key.a = sendToApp;
			_Browser_window.addEventListener('popstate', key);
			_Browser_window.navigator.userAgent.indexOf('Trident') < 0 || _Browser_window.addEventListener('hashchange', key);

			return F2(function(domNode, event)
			{
				if (!event.ctrlKey && !event.metaKey && !event.shiftKey && event.button < 1 && !domNode.target && !domNode.hasAttribute('download'))
				{
					event.preventDefault();
					var href = domNode.href;
					var curr = _Browser_getUrl();
					var next = $elm$url$Url$fromString(href).a;
					sendToApp(onUrlRequest(
						(next
							&& curr.protocol === next.protocol
							&& curr.host === next.host
							&& curr.port_.a === next.port_.a
						)
							? $elm$browser$Browser$Internal(next)
							: $elm$browser$Browser$External(href)
					));
				}
			});
		},
		init: function(flags)
		{
			return A3(impl.init, flags, _Browser_getUrl(), key);
		},
		view: impl.view,
		update: impl.update,
		subscriptions: impl.subscriptions
	});
}

function _Browser_getUrl()
{
	return $elm$url$Url$fromString(_VirtualDom_doc.location.href).a || _Debug_crash(1);
}

var _Browser_go = F2(function(key, n)
{
	return A2($elm$core$Task$perform, $elm$core$Basics$never, _Scheduler_binding(function() {
		n && history.go(n);
		key();
	}));
});

var _Browser_pushUrl = F2(function(key, url)
{
	return A2($elm$core$Task$perform, $elm$core$Basics$never, _Scheduler_binding(function() {
		history.pushState({}, '', url);
		key();
	}));
});

var _Browser_replaceUrl = F2(function(key, url)
{
	return A2($elm$core$Task$perform, $elm$core$Basics$never, _Scheduler_binding(function() {
		history.replaceState({}, '', url);
		key();
	}));
});



// GLOBAL EVENTS


var _Browser_fakeNode = { addEventListener: function() {}, removeEventListener: function() {} };
var _Browser_doc = typeof document !== 'undefined' ? document : _Browser_fakeNode;
var _Browser_window = typeof window !== 'undefined' ? window : _Browser_fakeNode;

var _Browser_on = F3(function(node, eventName, sendToSelf)
{
	return _Scheduler_spawn(_Scheduler_binding(function(callback)
	{
		function handler(event)	{ _Scheduler_rawSpawn(sendToSelf(event)); }
		node.addEventListener(eventName, handler, _VirtualDom_passiveSupported && { passive: true });
		return function() { node.removeEventListener(eventName, handler); };
	}));
});

var _Browser_decodeEvent = F2(function(decoder, event)
{
	var result = _Json_runHelp(decoder, event);
	return $elm$core$Result$isOk(result) ? $elm$core$Maybe$Just(result.a) : $elm$core$Maybe$Nothing;
});



// PAGE VISIBILITY


function _Browser_visibilityInfo()
{
	return (typeof _VirtualDom_doc.hidden !== 'undefined')
		? { hidden: 'hidden', change: 'visibilitychange' }
		:
	(typeof _VirtualDom_doc.mozHidden !== 'undefined')
		? { hidden: 'mozHidden', change: 'mozvisibilitychange' }
		:
	(typeof _VirtualDom_doc.msHidden !== 'undefined')
		? { hidden: 'msHidden', change: 'msvisibilitychange' }
		:
	(typeof _VirtualDom_doc.webkitHidden !== 'undefined')
		? { hidden: 'webkitHidden', change: 'webkitvisibilitychange' }
		: { hidden: 'hidden', change: 'visibilitychange' };
}



// ANIMATION FRAMES


function _Browser_rAF()
{
	return _Scheduler_binding(function(callback)
	{
		var id = _Browser_requestAnimationFrame(function() {
			callback(_Scheduler_succeed(Date.now()));
		});

		return function() {
			_Browser_cancelAnimationFrame(id);
		};
	});
}


function _Browser_now()
{
	return _Scheduler_binding(function(callback)
	{
		callback(_Scheduler_succeed(Date.now()));
	});
}



// DOM STUFF


function _Browser_withNode(id, doStuff)
{
	return _Scheduler_binding(function(callback)
	{
		_Browser_requestAnimationFrame(function() {
			var node = document.getElementById(id);
			callback(node
				? _Scheduler_succeed(doStuff(node))
				: _Scheduler_fail($elm$browser$Browser$Dom$NotFound(id))
			);
		});
	});
}


function _Browser_withWindow(doStuff)
{
	return _Scheduler_binding(function(callback)
	{
		_Browser_requestAnimationFrame(function() {
			callback(_Scheduler_succeed(doStuff()));
		});
	});
}


// FOCUS and BLUR


var _Browser_call = F2(function(functionName, id)
{
	return _Browser_withNode(id, function(node) {
		node[functionName]();
		return _Utils_Tuple0;
	});
});



// WINDOW VIEWPORT


function _Browser_getViewport()
{
	return {
		scene: _Browser_getScene(),
		viewport: {
			x: _Browser_window.pageXOffset,
			y: _Browser_window.pageYOffset,
			width: _Browser_doc.documentElement.clientWidth,
			height: _Browser_doc.documentElement.clientHeight
		}
	};
}

function _Browser_getScene()
{
	var body = _Browser_doc.body;
	var elem = _Browser_doc.documentElement;
	return {
		width: Math.max(body.scrollWidth, body.offsetWidth, elem.scrollWidth, elem.offsetWidth, elem.clientWidth),
		height: Math.max(body.scrollHeight, body.offsetHeight, elem.scrollHeight, elem.offsetHeight, elem.clientHeight)
	};
}

var _Browser_setViewport = F2(function(x, y)
{
	return _Browser_withWindow(function()
	{
		_Browser_window.scroll(x, y);
		return _Utils_Tuple0;
	});
});



// ELEMENT VIEWPORT


function _Browser_getViewportOf(id)
{
	return _Browser_withNode(id, function(node)
	{
		return {
			scene: {
				width: node.scrollWidth,
				height: node.scrollHeight
			},
			viewport: {
				x: node.scrollLeft,
				y: node.scrollTop,
				width: node.clientWidth,
				height: node.clientHeight
			}
		};
	});
}


var _Browser_setViewportOf = F3(function(id, x, y)
{
	return _Browser_withNode(id, function(node)
	{
		node.scrollLeft = x;
		node.scrollTop = y;
		return _Utils_Tuple0;
	});
});



// ELEMENT


function _Browser_getElement(id)
{
	return _Browser_withNode(id, function(node)
	{
		var rect = node.getBoundingClientRect();
		var x = _Browser_window.pageXOffset;
		var y = _Browser_window.pageYOffset;
		return {
			scene: _Browser_getScene(),
			viewport: {
				x: x,
				y: y,
				width: _Browser_doc.documentElement.clientWidth,
				height: _Browser_doc.documentElement.clientHeight
			},
			element: {
				x: x + rect.left,
				y: y + rect.top,
				width: rect.width,
				height: rect.height
			}
		};
	});
}



// LOAD and RELOAD


function _Browser_reload(skipCache)
{
	return A2($elm$core$Task$perform, $elm$core$Basics$never, _Scheduler_binding(function(callback)
	{
		_VirtualDom_doc.location.reload(skipCache);
	}));
}

function _Browser_load(url)
{
	return A2($elm$core$Task$perform, $elm$core$Basics$never, _Scheduler_binding(function(callback)
	{
		try
		{
			_Browser_window.location = url;
		}
		catch(err)
		{
			// Only Firefox can throw a NS_ERROR_MALFORMED_URI exception here.
			// Other browsers reload the page, so let's be consistent about that.
			_VirtualDom_doc.location.reload(false);
		}
	}));
}



// SEND REQUEST

var _Http_toTask = F3(function(router, toTask, request)
{
	return _Scheduler_binding(function(callback)
	{
		function done(response) {
			callback(toTask(request.expect.a(response)));
		}

		var xhr = new XMLHttpRequest();
		xhr.addEventListener('error', function() { done($elm$http$Http$NetworkError_); });
		xhr.addEventListener('timeout', function() { done($elm$http$Http$Timeout_); });
		xhr.addEventListener('load', function() { done(_Http_toResponse(request.expect.b, xhr)); });
		$elm$core$Maybe$isJust(request.tracker) && _Http_track(router, xhr, request.tracker.a);

		try {
			xhr.open(request.method, request.url, true);
		} catch (e) {
			return done($elm$http$Http$BadUrl_(request.url));
		}

		_Http_configureRequest(xhr, request);

		request.body.a && xhr.setRequestHeader('Content-Type', request.body.a);
		xhr.send(request.body.b);

		return function() { xhr.c = true; xhr.abort(); };
	});
});


// CONFIGURE

function _Http_configureRequest(xhr, request)
{
	for (var headers = request.headers; headers.b; headers = headers.b) // WHILE_CONS
	{
		xhr.setRequestHeader(headers.a.a, headers.a.b);
	}
	xhr.timeout = request.timeout.a || 0;
	xhr.responseType = request.expect.d;
	xhr.withCredentials = request.allowCookiesFromOtherDomains;
}


// RESPONSES

function _Http_toResponse(toBody, xhr)
{
	return A2(
		200 <= xhr.status && xhr.status < 300 ? $elm$http$Http$GoodStatus_ : $elm$http$Http$BadStatus_,
		_Http_toMetadata(xhr),
		toBody(xhr.response)
	);
}


// METADATA

function _Http_toMetadata(xhr)
{
	return {
		url: xhr.responseURL,
		statusCode: xhr.status,
		statusText: xhr.statusText,
		headers: _Http_parseHeaders(xhr.getAllResponseHeaders())
	};
}


// HEADERS

function _Http_parseHeaders(rawHeaders)
{
	if (!rawHeaders)
	{
		return $elm$core$Dict$empty;
	}

	var headers = $elm$core$Dict$empty;
	var headerPairs = rawHeaders.split('\r\n');
	for (var i = headerPairs.length; i--; )
	{
		var headerPair = headerPairs[i];
		var index = headerPair.indexOf(': ');
		if (index > 0)
		{
			var key = headerPair.substring(0, index);
			var value = headerPair.substring(index + 2);

			headers = A3($elm$core$Dict$update, key, function(oldValue) {
				return $elm$core$Maybe$Just($elm$core$Maybe$isJust(oldValue)
					? value + ', ' + oldValue.a
					: value
				);
			}, headers);
		}
	}
	return headers;
}


// EXPECT

var _Http_expect = F3(function(type, toBody, toValue)
{
	return {
		$: 0,
		d: type,
		b: toBody,
		a: toValue
	};
});

var _Http_mapExpect = F2(function(func, expect)
{
	return {
		$: 0,
		d: expect.d,
		b: expect.b,
		a: function(x) { return func(expect.a(x)); }
	};
});

function _Http_toDataView(arrayBuffer)
{
	return new DataView(arrayBuffer);
}


// BODY and PARTS

var _Http_emptyBody = { $: 0 };
var _Http_pair = F2(function(a, b) { return { $: 0, a: a, b: b }; });

function _Http_toFormData(parts)
{
	for (var formData = new FormData(); parts.b; parts = parts.b) // WHILE_CONS
	{
		var part = parts.a;
		formData.append(part.a, part.b);
	}
	return formData;
}

var _Http_bytesToBlob = F2(function(mime, bytes)
{
	return new Blob([bytes], { type: mime });
});


// PROGRESS

function _Http_track(router, xhr, tracker)
{
	// TODO check out lengthComputable on loadstart event

	xhr.upload.addEventListener('progress', function(event) {
		if (xhr.c) { return; }
		_Scheduler_rawSpawn(A2($elm$core$Platform$sendToSelf, router, _Utils_Tuple2(tracker, $elm$http$Http$Sending({
			sent: event.loaded,
			size: event.total
		}))));
	});
	xhr.addEventListener('progress', function(event) {
		if (xhr.c) { return; }
		_Scheduler_rawSpawn(A2($elm$core$Platform$sendToSelf, router, _Utils_Tuple2(tracker, $elm$http$Http$Receiving({
			received: event.loaded,
			size: event.lengthComputable ? $elm$core$Maybe$Just(event.total) : $elm$core$Maybe$Nothing
		}))));
	});
}

function _Url_percentEncode(string)
{
	return encodeURIComponent(string);
}

function _Url_percentDecode(string)
{
	try
	{
		return $elm$core$Maybe$Just(decodeURIComponent(string));
	}
	catch (e)
	{
		return $elm$core$Maybe$Nothing;
	}
}


// DECODER

var _File_decoder = _Json_decodePrim(function(value) {
	// NOTE: checks if `File` exists in case this is run on node
	return (typeof File !== 'undefined' && value instanceof File)
		? $elm$core$Result$Ok(value)
		: _Json_expecting('a FILE', value);
});


// METADATA

function _File_name(file) { return file.name; }
function _File_mime(file) { return file.type; }
function _File_size(file) { return file.size; }

function _File_lastModified(file)
{
	return $elm$time$Time$millisToPosix(file.lastModified);
}


// DOWNLOAD

var _File_downloadNode;

function _File_getDownloadNode()
{
	return _File_downloadNode || (_File_downloadNode = document.createElement('a'));
}

var _File_download = F3(function(name, mime, content)
{
	return _Scheduler_binding(function(callback)
	{
		var blob = new Blob([content], {type: mime});

		// for IE10+
		if (navigator.msSaveOrOpenBlob)
		{
			navigator.msSaveOrOpenBlob(blob, name);
			return;
		}

		// for HTML5
		var node = _File_getDownloadNode();
		var objectUrl = URL.createObjectURL(blob);
		node.href = objectUrl;
		node.download = name;
		_File_click(node);
		URL.revokeObjectURL(objectUrl);
	});
});

function _File_downloadUrl(href)
{
	return _Scheduler_binding(function(callback)
	{
		var node = _File_getDownloadNode();
		node.href = href;
		node.download = '';
		node.origin === location.origin || (node.target = '_blank');
		_File_click(node);
	});
}


// IE COMPATIBILITY

function _File_makeBytesSafeForInternetExplorer(bytes)
{
	// only needed by IE10 and IE11 to fix https://github.com/elm/file/issues/10
	// all other browsers can just run `new Blob([bytes])` directly with no problem
	//
	return new Uint8Array(bytes.buffer, bytes.byteOffset, bytes.byteLength);
}

function _File_click(node)
{
	// only needed by IE10 and IE11 to fix https://github.com/elm/file/issues/11
	// all other browsers have MouseEvent and do not need this conditional stuff
	//
	if (typeof MouseEvent === 'function')
	{
		node.dispatchEvent(new MouseEvent('click'));
	}
	else
	{
		var event = document.createEvent('MouseEvents');
		event.initMouseEvent('click', true, true, window, 0, 0, 0, 0, 0, false, false, false, false, 0, null);
		document.body.appendChild(node);
		node.dispatchEvent(event);
		document.body.removeChild(node);
	}
}


// UPLOAD

var _File_node;

function _File_uploadOne(mimes)
{
	return _Scheduler_binding(function(callback)
	{
		_File_node = document.createElement('input');
		_File_node.type = 'file';
		_File_node.accept = A2($elm$core$String$join, ',', mimes);
		_File_node.addEventListener('change', function(event)
		{
			callback(_Scheduler_succeed(event.target.files[0]));
		});
		_File_click(_File_node);
	});
}

function _File_uploadOneOrMore(mimes)
{
	return _Scheduler_binding(function(callback)
	{
		_File_node = document.createElement('input');
		_File_node.type = 'file';
		_File_node.multiple = true;
		_File_node.accept = A2($elm$core$String$join, ',', mimes);
		_File_node.addEventListener('change', function(event)
		{
			var elmFiles = _List_fromArray(event.target.files);
			callback(_Scheduler_succeed(_Utils_Tuple2(elmFiles.a, elmFiles.b)));
		});
		_File_click(_File_node);
	});
}


// CONTENT

function _File_toString(blob)
{
	return _Scheduler_binding(function(callback)
	{
		var reader = new FileReader();
		reader.addEventListener('loadend', function() {
			callback(_Scheduler_succeed(reader.result));
		});
		reader.readAsText(blob);
		return function() { reader.abort(); };
	});
}

function _File_toBytes(blob)
{
	return _Scheduler_binding(function(callback)
	{
		var reader = new FileReader();
		reader.addEventListener('loadend', function() {
			callback(_Scheduler_succeed(new DataView(reader.result)));
		});
		reader.readAsArrayBuffer(blob);
		return function() { reader.abort(); };
	});
}

function _File_toUrl(blob)
{
	return _Scheduler_binding(function(callback)
	{
		var reader = new FileReader();
		reader.addEventListener('loadend', function() {
			callback(_Scheduler_succeed(reader.result));
		});
		reader.readAsDataURL(blob);
		return function() { reader.abort(); };
	});
}

var $elm$core$Basics$EQ = {$: 'EQ'};
var $elm$core$Basics$LT = {$: 'LT'};
var $elm$core$List$cons = _List_cons;
var $elm$core$Elm$JsArray$foldr = _JsArray_foldr;
var $elm$core$Array$foldr = F3(
	function (func, baseCase, _v0) {
		var tree = _v0.c;
		var tail = _v0.d;
		var helper = F2(
			function (node, acc) {
				if (node.$ === 'SubTree') {
					var subTree = node.a;
					return A3($elm$core$Elm$JsArray$foldr, helper, acc, subTree);
				} else {
					var values = node.a;
					return A3($elm$core$Elm$JsArray$foldr, func, acc, values);
				}
			});
		return A3(
			$elm$core$Elm$JsArray$foldr,
			helper,
			A3($elm$core$Elm$JsArray$foldr, func, baseCase, tail),
			tree);
	});
var $elm$core$Array$toList = function (array) {
	return A3($elm$core$Array$foldr, $elm$core$List$cons, _List_Nil, array);
};
var $elm$core$Dict$foldr = F3(
	function (func, acc, t) {
		foldr:
		while (true) {
			if (t.$ === 'RBEmpty_elm_builtin') {
				return acc;
			} else {
				var key = t.b;
				var value = t.c;
				var left = t.d;
				var right = t.e;
				var $temp$func = func,
					$temp$acc = A3(
					func,
					key,
					value,
					A3($elm$core$Dict$foldr, func, acc, right)),
					$temp$t = left;
				func = $temp$func;
				acc = $temp$acc;
				t = $temp$t;
				continue foldr;
			}
		}
	});
var $elm$core$Dict$toList = function (dict) {
	return A3(
		$elm$core$Dict$foldr,
		F3(
			function (key, value, list) {
				return A2(
					$elm$core$List$cons,
					_Utils_Tuple2(key, value),
					list);
			}),
		_List_Nil,
		dict);
};
var $elm$core$Dict$keys = function (dict) {
	return A3(
		$elm$core$Dict$foldr,
		F3(
			function (key, value, keyList) {
				return A2($elm$core$List$cons, key, keyList);
			}),
		_List_Nil,
		dict);
};
var $elm$core$Set$toList = function (_v0) {
	var dict = _v0.a;
	return $elm$core$Dict$keys(dict);
};
var $elm$core$Basics$GT = {$: 'GT'};
var $author$project$Sharecrop$Types$LinkClicked = function (a) {
	return {$: 'LinkClicked', a: a};
};
var $author$project$Sharecrop$Types$UrlChanged = function (a) {
	return {$: 'UrlChanged', a: a};
};
var $elm$core$Result$Err = function (a) {
	return {$: 'Err', a: a};
};
var $elm$json$Json$Decode$Failure = F2(
	function (a, b) {
		return {$: 'Failure', a: a, b: b};
	});
var $elm$json$Json$Decode$Field = F2(
	function (a, b) {
		return {$: 'Field', a: a, b: b};
	});
var $elm$json$Json$Decode$Index = F2(
	function (a, b) {
		return {$: 'Index', a: a, b: b};
	});
var $elm$core$Result$Ok = function (a) {
	return {$: 'Ok', a: a};
};
var $elm$json$Json$Decode$OneOf = function (a) {
	return {$: 'OneOf', a: a};
};
var $elm$core$Basics$False = {$: 'False'};
var $elm$core$Basics$add = _Basics_add;
var $elm$core$Maybe$Just = function (a) {
	return {$: 'Just', a: a};
};
var $elm$core$Maybe$Nothing = {$: 'Nothing'};
var $elm$core$String$all = _String_all;
var $elm$core$Basics$and = _Basics_and;
var $elm$core$Basics$append = _Utils_append;
var $elm$json$Json$Encode$encode = _Json_encode;
var $elm$core$String$fromInt = _String_fromNumber;
var $elm$core$String$join = F2(
	function (sep, chunks) {
		return A2(
			_String_join,
			sep,
			_List_toArray(chunks));
	});
var $elm$core$String$split = F2(
	function (sep, string) {
		return _List_fromArray(
			A2(_String_split, sep, string));
	});
var $elm$json$Json$Decode$indent = function (str) {
	return A2(
		$elm$core$String$join,
		'\n    ',
		A2($elm$core$String$split, '\n', str));
};
var $elm$core$List$foldl = F3(
	function (func, acc, list) {
		foldl:
		while (true) {
			if (!list.b) {
				return acc;
			} else {
				var x = list.a;
				var xs = list.b;
				var $temp$func = func,
					$temp$acc = A2(func, x, acc),
					$temp$list = xs;
				func = $temp$func;
				acc = $temp$acc;
				list = $temp$list;
				continue foldl;
			}
		}
	});
var $elm$core$List$length = function (xs) {
	return A3(
		$elm$core$List$foldl,
		F2(
			function (_v0, i) {
				return i + 1;
			}),
		0,
		xs);
};
var $elm$core$List$map2 = _List_map2;
var $elm$core$Basics$le = _Utils_le;
var $elm$core$Basics$sub = _Basics_sub;
var $elm$core$List$rangeHelp = F3(
	function (lo, hi, list) {
		rangeHelp:
		while (true) {
			if (_Utils_cmp(lo, hi) < 1) {
				var $temp$lo = lo,
					$temp$hi = hi - 1,
					$temp$list = A2($elm$core$List$cons, hi, list);
				lo = $temp$lo;
				hi = $temp$hi;
				list = $temp$list;
				continue rangeHelp;
			} else {
				return list;
			}
		}
	});
var $elm$core$List$range = F2(
	function (lo, hi) {
		return A3($elm$core$List$rangeHelp, lo, hi, _List_Nil);
	});
var $elm$core$List$indexedMap = F2(
	function (f, xs) {
		return A3(
			$elm$core$List$map2,
			f,
			A2(
				$elm$core$List$range,
				0,
				$elm$core$List$length(xs) - 1),
			xs);
	});
var $elm$core$Char$toCode = _Char_toCode;
var $elm$core$Char$isLower = function (_char) {
	var code = $elm$core$Char$toCode(_char);
	return (97 <= code) && (code <= 122);
};
var $elm$core$Char$isUpper = function (_char) {
	var code = $elm$core$Char$toCode(_char);
	return (code <= 90) && (65 <= code);
};
var $elm$core$Basics$or = _Basics_or;
var $elm$core$Char$isAlpha = function (_char) {
	return $elm$core$Char$isLower(_char) || $elm$core$Char$isUpper(_char);
};
var $elm$core$Char$isDigit = function (_char) {
	var code = $elm$core$Char$toCode(_char);
	return (code <= 57) && (48 <= code);
};
var $elm$core$Char$isAlphaNum = function (_char) {
	return $elm$core$Char$isLower(_char) || ($elm$core$Char$isUpper(_char) || $elm$core$Char$isDigit(_char));
};
var $elm$core$List$reverse = function (list) {
	return A3($elm$core$List$foldl, $elm$core$List$cons, _List_Nil, list);
};
var $elm$core$String$uncons = _String_uncons;
var $elm$json$Json$Decode$errorOneOf = F2(
	function (i, error) {
		return '\n\n(' + ($elm$core$String$fromInt(i + 1) + (') ' + $elm$json$Json$Decode$indent(
			$elm$json$Json$Decode$errorToString(error))));
	});
var $elm$json$Json$Decode$errorToString = function (error) {
	return A2($elm$json$Json$Decode$errorToStringHelp, error, _List_Nil);
};
var $elm$json$Json$Decode$errorToStringHelp = F2(
	function (error, context) {
		errorToStringHelp:
		while (true) {
			switch (error.$) {
				case 'Field':
					var f = error.a;
					var err = error.b;
					var isSimple = function () {
						var _v1 = $elm$core$String$uncons(f);
						if (_v1.$ === 'Nothing') {
							return false;
						} else {
							var _v2 = _v1.a;
							var _char = _v2.a;
							var rest = _v2.b;
							return $elm$core$Char$isAlpha(_char) && A2($elm$core$String$all, $elm$core$Char$isAlphaNum, rest);
						}
					}();
					var fieldName = isSimple ? ('.' + f) : ('[\'' + (f + '\']'));
					var $temp$error = err,
						$temp$context = A2($elm$core$List$cons, fieldName, context);
					error = $temp$error;
					context = $temp$context;
					continue errorToStringHelp;
				case 'Index':
					var i = error.a;
					var err = error.b;
					var indexName = '[' + ($elm$core$String$fromInt(i) + ']');
					var $temp$error = err,
						$temp$context = A2($elm$core$List$cons, indexName, context);
					error = $temp$error;
					context = $temp$context;
					continue errorToStringHelp;
				case 'OneOf':
					var errors = error.a;
					if (!errors.b) {
						return 'Ran into a Json.Decode.oneOf with no possibilities' + function () {
							if (!context.b) {
								return '!';
							} else {
								return ' at json' + A2(
									$elm$core$String$join,
									'',
									$elm$core$List$reverse(context));
							}
						}();
					} else {
						if (!errors.b.b) {
							var err = errors.a;
							var $temp$error = err,
								$temp$context = context;
							error = $temp$error;
							context = $temp$context;
							continue errorToStringHelp;
						} else {
							var starter = function () {
								if (!context.b) {
									return 'Json.Decode.oneOf';
								} else {
									return 'The Json.Decode.oneOf at json' + A2(
										$elm$core$String$join,
										'',
										$elm$core$List$reverse(context));
								}
							}();
							var introduction = starter + (' failed in the following ' + ($elm$core$String$fromInt(
								$elm$core$List$length(errors)) + ' ways:'));
							return A2(
								$elm$core$String$join,
								'\n\n',
								A2(
									$elm$core$List$cons,
									introduction,
									A2($elm$core$List$indexedMap, $elm$json$Json$Decode$errorOneOf, errors)));
						}
					}
				default:
					var msg = error.a;
					var json = error.b;
					var introduction = function () {
						if (!context.b) {
							return 'Problem with the given value:\n\n';
						} else {
							return 'Problem with the value at json' + (A2(
								$elm$core$String$join,
								'',
								$elm$core$List$reverse(context)) + ':\n\n    ');
						}
					}();
					return introduction + ($elm$json$Json$Decode$indent(
						A2($elm$json$Json$Encode$encode, 4, json)) + ('\n\n' + msg));
			}
		}
	});
var $elm$core$Array$branchFactor = 32;
var $elm$core$Array$Array_elm_builtin = F4(
	function (a, b, c, d) {
		return {$: 'Array_elm_builtin', a: a, b: b, c: c, d: d};
	});
var $elm$core$Elm$JsArray$empty = _JsArray_empty;
var $elm$core$Basics$ceiling = _Basics_ceiling;
var $elm$core$Basics$fdiv = _Basics_fdiv;
var $elm$core$Basics$logBase = F2(
	function (base, number) {
		return _Basics_log(number) / _Basics_log(base);
	});
var $elm$core$Basics$toFloat = _Basics_toFloat;
var $elm$core$Array$shiftStep = $elm$core$Basics$ceiling(
	A2($elm$core$Basics$logBase, 2, $elm$core$Array$branchFactor));
var $elm$core$Array$empty = A4($elm$core$Array$Array_elm_builtin, 0, $elm$core$Array$shiftStep, $elm$core$Elm$JsArray$empty, $elm$core$Elm$JsArray$empty);
var $elm$core$Elm$JsArray$initialize = _JsArray_initialize;
var $elm$core$Array$Leaf = function (a) {
	return {$: 'Leaf', a: a};
};
var $elm$core$Basics$apL = F2(
	function (f, x) {
		return f(x);
	});
var $elm$core$Basics$apR = F2(
	function (x, f) {
		return f(x);
	});
var $elm$core$Basics$eq = _Utils_equal;
var $elm$core$Basics$floor = _Basics_floor;
var $elm$core$Elm$JsArray$length = _JsArray_length;
var $elm$core$Basics$gt = _Utils_gt;
var $elm$core$Basics$max = F2(
	function (x, y) {
		return (_Utils_cmp(x, y) > 0) ? x : y;
	});
var $elm$core$Basics$mul = _Basics_mul;
var $elm$core$Array$SubTree = function (a) {
	return {$: 'SubTree', a: a};
};
var $elm$core$Elm$JsArray$initializeFromList = _JsArray_initializeFromList;
var $elm$core$Array$compressNodes = F2(
	function (nodes, acc) {
		compressNodes:
		while (true) {
			var _v0 = A2($elm$core$Elm$JsArray$initializeFromList, $elm$core$Array$branchFactor, nodes);
			var node = _v0.a;
			var remainingNodes = _v0.b;
			var newAcc = A2(
				$elm$core$List$cons,
				$elm$core$Array$SubTree(node),
				acc);
			if (!remainingNodes.b) {
				return $elm$core$List$reverse(newAcc);
			} else {
				var $temp$nodes = remainingNodes,
					$temp$acc = newAcc;
				nodes = $temp$nodes;
				acc = $temp$acc;
				continue compressNodes;
			}
		}
	});
var $elm$core$Tuple$first = function (_v0) {
	var x = _v0.a;
	return x;
};
var $elm$core$Array$treeFromBuilder = F2(
	function (nodeList, nodeListSize) {
		treeFromBuilder:
		while (true) {
			var newNodeSize = $elm$core$Basics$ceiling(nodeListSize / $elm$core$Array$branchFactor);
			if (newNodeSize === 1) {
				return A2($elm$core$Elm$JsArray$initializeFromList, $elm$core$Array$branchFactor, nodeList).a;
			} else {
				var $temp$nodeList = A2($elm$core$Array$compressNodes, nodeList, _List_Nil),
					$temp$nodeListSize = newNodeSize;
				nodeList = $temp$nodeList;
				nodeListSize = $temp$nodeListSize;
				continue treeFromBuilder;
			}
		}
	});
var $elm$core$Array$builderToArray = F2(
	function (reverseNodeList, builder) {
		if (!builder.nodeListSize) {
			return A4(
				$elm$core$Array$Array_elm_builtin,
				$elm$core$Elm$JsArray$length(builder.tail),
				$elm$core$Array$shiftStep,
				$elm$core$Elm$JsArray$empty,
				builder.tail);
		} else {
			var treeLen = builder.nodeListSize * $elm$core$Array$branchFactor;
			var depth = $elm$core$Basics$floor(
				A2($elm$core$Basics$logBase, $elm$core$Array$branchFactor, treeLen - 1));
			var correctNodeList = reverseNodeList ? $elm$core$List$reverse(builder.nodeList) : builder.nodeList;
			var tree = A2($elm$core$Array$treeFromBuilder, correctNodeList, builder.nodeListSize);
			return A4(
				$elm$core$Array$Array_elm_builtin,
				$elm$core$Elm$JsArray$length(builder.tail) + treeLen,
				A2($elm$core$Basics$max, 5, depth * $elm$core$Array$shiftStep),
				tree,
				builder.tail);
		}
	});
var $elm$core$Basics$idiv = _Basics_idiv;
var $elm$core$Basics$lt = _Utils_lt;
var $elm$core$Array$initializeHelp = F5(
	function (fn, fromIndex, len, nodeList, tail) {
		initializeHelp:
		while (true) {
			if (fromIndex < 0) {
				return A2(
					$elm$core$Array$builderToArray,
					false,
					{nodeList: nodeList, nodeListSize: (len / $elm$core$Array$branchFactor) | 0, tail: tail});
			} else {
				var leaf = $elm$core$Array$Leaf(
					A3($elm$core$Elm$JsArray$initialize, $elm$core$Array$branchFactor, fromIndex, fn));
				var $temp$fn = fn,
					$temp$fromIndex = fromIndex - $elm$core$Array$branchFactor,
					$temp$len = len,
					$temp$nodeList = A2($elm$core$List$cons, leaf, nodeList),
					$temp$tail = tail;
				fn = $temp$fn;
				fromIndex = $temp$fromIndex;
				len = $temp$len;
				nodeList = $temp$nodeList;
				tail = $temp$tail;
				continue initializeHelp;
			}
		}
	});
var $elm$core$Basics$remainderBy = _Basics_remainderBy;
var $elm$core$Array$initialize = F2(
	function (len, fn) {
		if (len <= 0) {
			return $elm$core$Array$empty;
		} else {
			var tailLen = len % $elm$core$Array$branchFactor;
			var tail = A3($elm$core$Elm$JsArray$initialize, tailLen, len - tailLen, fn);
			var initialFromIndex = (len - tailLen) - $elm$core$Array$branchFactor;
			return A5($elm$core$Array$initializeHelp, fn, initialFromIndex, len, _List_Nil, tail);
		}
	});
var $elm$core$Basics$True = {$: 'True'};
var $elm$core$Result$isOk = function (result) {
	if (result.$ === 'Ok') {
		return true;
	} else {
		return false;
	}
};
var $elm$json$Json$Decode$andThen = _Json_andThen;
var $elm$json$Json$Decode$map = _Json_map1;
var $elm$json$Json$Decode$map2 = _Json_map2;
var $elm$json$Json$Decode$succeed = _Json_succeed;
var $elm$virtual_dom$VirtualDom$toHandlerInt = function (handler) {
	switch (handler.$) {
		case 'Normal':
			return 0;
		case 'MayStopPropagation':
			return 1;
		case 'MayPreventDefault':
			return 2;
		default:
			return 3;
	}
};
var $elm$browser$Browser$External = function (a) {
	return {$: 'External', a: a};
};
var $elm$browser$Browser$Internal = function (a) {
	return {$: 'Internal', a: a};
};
var $elm$core$Basics$identity = function (x) {
	return x;
};
var $elm$browser$Browser$Dom$NotFound = function (a) {
	return {$: 'NotFound', a: a};
};
var $elm$url$Url$Http = {$: 'Http'};
var $elm$url$Url$Https = {$: 'Https'};
var $elm$url$Url$Url = F6(
	function (protocol, host, port_, path, query, fragment) {
		return {fragment: fragment, host: host, path: path, port_: port_, protocol: protocol, query: query};
	});
var $elm$core$String$contains = _String_contains;
var $elm$core$String$length = _String_length;
var $elm$core$String$slice = _String_slice;
var $elm$core$String$dropLeft = F2(
	function (n, string) {
		return (n < 1) ? string : A3(
			$elm$core$String$slice,
			n,
			$elm$core$String$length(string),
			string);
	});
var $elm$core$String$indexes = _String_indexes;
var $elm$core$String$isEmpty = function (string) {
	return string === '';
};
var $elm$core$String$left = F2(
	function (n, string) {
		return (n < 1) ? '' : A3($elm$core$String$slice, 0, n, string);
	});
var $elm$core$String$toInt = _String_toInt;
var $elm$url$Url$chompBeforePath = F5(
	function (protocol, path, params, frag, str) {
		if ($elm$core$String$isEmpty(str) || A2($elm$core$String$contains, '@', str)) {
			return $elm$core$Maybe$Nothing;
		} else {
			var _v0 = A2($elm$core$String$indexes, ':', str);
			if (!_v0.b) {
				return $elm$core$Maybe$Just(
					A6($elm$url$Url$Url, protocol, str, $elm$core$Maybe$Nothing, path, params, frag));
			} else {
				if (!_v0.b.b) {
					var i = _v0.a;
					var _v1 = $elm$core$String$toInt(
						A2($elm$core$String$dropLeft, i + 1, str));
					if (_v1.$ === 'Nothing') {
						return $elm$core$Maybe$Nothing;
					} else {
						var port_ = _v1;
						return $elm$core$Maybe$Just(
							A6(
								$elm$url$Url$Url,
								protocol,
								A2($elm$core$String$left, i, str),
								port_,
								path,
								params,
								frag));
					}
				} else {
					return $elm$core$Maybe$Nothing;
				}
			}
		}
	});
var $elm$url$Url$chompBeforeQuery = F4(
	function (protocol, params, frag, str) {
		if ($elm$core$String$isEmpty(str)) {
			return $elm$core$Maybe$Nothing;
		} else {
			var _v0 = A2($elm$core$String$indexes, '/', str);
			if (!_v0.b) {
				return A5($elm$url$Url$chompBeforePath, protocol, '/', params, frag, str);
			} else {
				var i = _v0.a;
				return A5(
					$elm$url$Url$chompBeforePath,
					protocol,
					A2($elm$core$String$dropLeft, i, str),
					params,
					frag,
					A2($elm$core$String$left, i, str));
			}
		}
	});
var $elm$url$Url$chompBeforeFragment = F3(
	function (protocol, frag, str) {
		if ($elm$core$String$isEmpty(str)) {
			return $elm$core$Maybe$Nothing;
		} else {
			var _v0 = A2($elm$core$String$indexes, '?', str);
			if (!_v0.b) {
				return A4($elm$url$Url$chompBeforeQuery, protocol, $elm$core$Maybe$Nothing, frag, str);
			} else {
				var i = _v0.a;
				return A4(
					$elm$url$Url$chompBeforeQuery,
					protocol,
					$elm$core$Maybe$Just(
						A2($elm$core$String$dropLeft, i + 1, str)),
					frag,
					A2($elm$core$String$left, i, str));
			}
		}
	});
var $elm$url$Url$chompAfterProtocol = F2(
	function (protocol, str) {
		if ($elm$core$String$isEmpty(str)) {
			return $elm$core$Maybe$Nothing;
		} else {
			var _v0 = A2($elm$core$String$indexes, '#', str);
			if (!_v0.b) {
				return A3($elm$url$Url$chompBeforeFragment, protocol, $elm$core$Maybe$Nothing, str);
			} else {
				var i = _v0.a;
				return A3(
					$elm$url$Url$chompBeforeFragment,
					protocol,
					$elm$core$Maybe$Just(
						A2($elm$core$String$dropLeft, i + 1, str)),
					A2($elm$core$String$left, i, str));
			}
		}
	});
var $elm$core$String$startsWith = _String_startsWith;
var $elm$url$Url$fromString = function (str) {
	return A2($elm$core$String$startsWith, 'http://', str) ? A2(
		$elm$url$Url$chompAfterProtocol,
		$elm$url$Url$Http,
		A2($elm$core$String$dropLeft, 7, str)) : (A2($elm$core$String$startsWith, 'https://', str) ? A2(
		$elm$url$Url$chompAfterProtocol,
		$elm$url$Url$Https,
		A2($elm$core$String$dropLeft, 8, str)) : $elm$core$Maybe$Nothing);
};
var $elm$core$Basics$never = function (_v0) {
	never:
	while (true) {
		var nvr = _v0.a;
		var $temp$_v0 = nvr;
		_v0 = $temp$_v0;
		continue never;
	}
};
var $elm$core$Task$Perform = function (a) {
	return {$: 'Perform', a: a};
};
var $elm$core$Task$succeed = _Scheduler_succeed;
var $elm$core$Task$init = $elm$core$Task$succeed(_Utils_Tuple0);
var $elm$core$List$foldrHelper = F4(
	function (fn, acc, ctr, ls) {
		if (!ls.b) {
			return acc;
		} else {
			var a = ls.a;
			var r1 = ls.b;
			if (!r1.b) {
				return A2(fn, a, acc);
			} else {
				var b = r1.a;
				var r2 = r1.b;
				if (!r2.b) {
					return A2(
						fn,
						a,
						A2(fn, b, acc));
				} else {
					var c = r2.a;
					var r3 = r2.b;
					if (!r3.b) {
						return A2(
							fn,
							a,
							A2(
								fn,
								b,
								A2(fn, c, acc)));
					} else {
						var d = r3.a;
						var r4 = r3.b;
						var res = (ctr > 500) ? A3(
							$elm$core$List$foldl,
							fn,
							acc,
							$elm$core$List$reverse(r4)) : A4($elm$core$List$foldrHelper, fn, acc, ctr + 1, r4);
						return A2(
							fn,
							a,
							A2(
								fn,
								b,
								A2(
									fn,
									c,
									A2(fn, d, res))));
					}
				}
			}
		}
	});
var $elm$core$List$foldr = F3(
	function (fn, acc, ls) {
		return A4($elm$core$List$foldrHelper, fn, acc, 0, ls);
	});
var $elm$core$List$map = F2(
	function (f, xs) {
		return A3(
			$elm$core$List$foldr,
			F2(
				function (x, acc) {
					return A2(
						$elm$core$List$cons,
						f(x),
						acc);
				}),
			_List_Nil,
			xs);
	});
var $elm$core$Task$andThen = _Scheduler_andThen;
var $elm$core$Task$map = F2(
	function (func, taskA) {
		return A2(
			$elm$core$Task$andThen,
			function (a) {
				return $elm$core$Task$succeed(
					func(a));
			},
			taskA);
	});
var $elm$core$Task$map2 = F3(
	function (func, taskA, taskB) {
		return A2(
			$elm$core$Task$andThen,
			function (a) {
				return A2(
					$elm$core$Task$andThen,
					function (b) {
						return $elm$core$Task$succeed(
							A2(func, a, b));
					},
					taskB);
			},
			taskA);
	});
var $elm$core$Task$sequence = function (tasks) {
	return A3(
		$elm$core$List$foldr,
		$elm$core$Task$map2($elm$core$List$cons),
		$elm$core$Task$succeed(_List_Nil),
		tasks);
};
var $elm$core$Platform$sendToApp = _Platform_sendToApp;
var $elm$core$Task$spawnCmd = F2(
	function (router, _v0) {
		var task = _v0.a;
		return _Scheduler_spawn(
			A2(
				$elm$core$Task$andThen,
				$elm$core$Platform$sendToApp(router),
				task));
	});
var $elm$core$Task$onEffects = F3(
	function (router, commands, state) {
		return A2(
			$elm$core$Task$map,
			function (_v0) {
				return _Utils_Tuple0;
			},
			$elm$core$Task$sequence(
				A2(
					$elm$core$List$map,
					$elm$core$Task$spawnCmd(router),
					commands)));
	});
var $elm$core$Task$onSelfMsg = F3(
	function (_v0, _v1, _v2) {
		return $elm$core$Task$succeed(_Utils_Tuple0);
	});
var $elm$core$Task$cmdMap = F2(
	function (tagger, _v0) {
		var task = _v0.a;
		return $elm$core$Task$Perform(
			A2($elm$core$Task$map, tagger, task));
	});
_Platform_effectManagers['Task'] = _Platform_createManager($elm$core$Task$init, $elm$core$Task$onEffects, $elm$core$Task$onSelfMsg, $elm$core$Task$cmdMap);
var $elm$core$Task$command = _Platform_leaf('Task');
var $elm$core$Task$perform = F2(
	function (toMessage, task) {
		return $elm$core$Task$command(
			$elm$core$Task$Perform(
				A2($elm$core$Task$map, toMessage, task)));
	});
var $elm$browser$Browser$application = _Browser_application;
var $elm$json$Json$Decode$bool = _Json_decodeBool;
var $elm$json$Json$Decode$field = _Json_decodeField;
var $author$project$Sharecrop$Types$LoggedOut = {$: 'LoggedOut'};
var $author$project$Sharecrop$Types$AdminPage = {$: 'AdminPage'};
var $author$project$Sharecrop$Types$AgentsPage = {$: 'AgentsPage'};
var $author$project$Sharecrop$Types$CollectibleDetailPage = function (a) {
	return {$: 'CollectibleDetailPage', a: a};
};
var $author$project$Sharecrop$Types$CollectiblesPage = {$: 'CollectiblesPage'};
var $author$project$Sharecrop$Types$CreateTaskPage = {$: 'CreateTaskPage'};
var $author$project$Sharecrop$Types$DiscoveryPage = {$: 'DiscoveryPage'};
var $author$project$Sharecrop$Types$FundingPage = {$: 'FundingPage'};
var $author$project$Sharecrop$Types$InboxPage = {$: 'InboxPage'};
var $author$project$Sharecrop$Types$NotFoundPage = {$: 'NotFoundPage'};
var $author$project$Sharecrop$Types$OrganizationDetailPage = function (a) {
	return {$: 'OrganizationDetailPage', a: a};
};
var $author$project$Sharecrop$Types$OrganizationsPage = {$: 'OrganizationsPage'};
var $author$project$Sharecrop$Types$OverviewPage = {$: 'OverviewPage'};
var $author$project$Sharecrop$Types$SeriesDetailPage = function (a) {
	return {$: 'SeriesDetailPage', a: a};
};
var $author$project$Sharecrop$Types$SeriesListPage = {$: 'SeriesListPage'};
var $author$project$Sharecrop$Types$TaskDetailPage = function (a) {
	return {$: 'TaskDetailPage', a: a};
};
var $author$project$Sharecrop$Types$TasksPage = {$: 'TasksPage'};
var $author$project$Sharecrop$Types$TeamDetailPage = function (a) {
	return {$: 'TeamDetailPage', a: a};
};
var $author$project$Sharecrop$Types$UserDetailPage = function (a) {
	return {$: 'UserDetailPage', a: a};
};
var $author$project$Sharecrop$Types$UserSubmissionsPage = function (a) {
	return {$: 'UserSubmissionsPage', a: a};
};
var $author$project$Sharecrop$Types$UserWorkPage = function (a) {
	return {$: 'UserWorkPage', a: a};
};
var $elm$core$Maybe$withDefault = F2(
	function (_default, maybe) {
		if (maybe.$ === 'Just') {
			var value = maybe.a;
			return value;
		} else {
			return _default;
		}
	});
var $author$project$Main$pageFromUrl = function (url) {
	var fragment = A2($elm$core$Maybe$withDefault, '', url.fragment);
	var _v0 = A2(
		$elm$core$String$split,
		'/',
		A2($elm$core$String$dropLeft, 1, fragment));
	_v0$19:
	while (true) {
		if (_v0.b) {
			if (!_v0.b.b) {
				switch (_v0.a) {
					case '':
						return $author$project$Sharecrop$Types$OverviewPage;
					case 'tasks':
						return $author$project$Sharecrop$Types$TasksPage;
					case 'discovery':
						return $author$project$Sharecrop$Types$DiscoveryPage;
					case 'funding':
						return $author$project$Sharecrop$Types$FundingPage;
					case 'agents':
						return $author$project$Sharecrop$Types$AgentsPage;
					case 'collectibles':
						return $author$project$Sharecrop$Types$CollectiblesPage;
					case 'series':
						return $author$project$Sharecrop$Types$SeriesListPage;
					case 'admin':
						return $author$project$Sharecrop$Types$AdminPage;
					case 'inbox':
						return $author$project$Sharecrop$Types$InboxPage;
					case 'organizations':
						return $author$project$Sharecrop$Types$OrganizationsPage;
					default:
						break _v0$19;
				}
			} else {
				if (!_v0.b.b.b) {
					switch (_v0.a) {
						case 'tasks':
							if (_v0.b.a === 'new') {
								var _v1 = _v0.b;
								return $author$project$Sharecrop$Types$CreateTaskPage;
							} else {
								var _v2 = _v0.b;
								var taskId = _v2.a;
								return $author$project$Sharecrop$Types$TaskDetailPage(taskId);
							}
						case 'collectibles':
							var _v3 = _v0.b;
							var collectibleId = _v3.a;
							return $author$project$Sharecrop$Types$CollectibleDetailPage(collectibleId);
						case 'series':
							var _v4 = _v0.b;
							var seriesId = _v4.a;
							return $author$project$Sharecrop$Types$SeriesDetailPage(seriesId);
						case 'teams':
							var _v5 = _v0.b;
							var teamId = _v5.a;
							return $author$project$Sharecrop$Types$TeamDetailPage(teamId);
						case 'organizations':
							var _v6 = _v0.b;
							var organizationId = _v6.a;
							return $author$project$Sharecrop$Types$OrganizationDetailPage(organizationId);
						case 'users':
							var _v7 = _v0.b;
							var userId = _v7.a;
							return $author$project$Sharecrop$Types$UserDetailPage(userId);
						default:
							break _v0$19;
					}
				} else {
					if ((_v0.a === 'users') && (!_v0.b.b.b.b)) {
						switch (_v0.b.b.a) {
							case 'work':
								var _v8 = _v0.b;
								var userId = _v8.a;
								var _v9 = _v8.b;
								return $author$project$Sharecrop$Types$UserWorkPage(userId);
							case 'submissions':
								var _v10 = _v0.b;
								var userId = _v10.a;
								var _v11 = _v10.b;
								return $author$project$Sharecrop$Types$UserSubmissionsPage(userId);
							default:
								break _v0$19;
						}
					} else {
						break _v0$19;
					}
				}
			}
		} else {
			break _v0$19;
		}
	}
	return $author$project$Sharecrop$Types$NotFoundPage;
};
var $author$project$Main$initialModel = F3(
	function (flags, key, url) {
		return {
			authError: $elm$core$Maybe$Nothing,
			demo: flags.demo,
			email: '',
			key: key,
			origin: flags.origin,
			password: '',
			resetEmail: '',
			resetPassword: '',
			resetToken: '',
			route: $author$project$Main$pageFromUrl(url),
			session: $author$project$Sharecrop$Types$LoggedOut
		};
	});
var $elm$core$Platform$Sub$batch = _Platform_batch;
var $elm$core$Platform$Sub$none = $elm$core$Platform$Sub$batch(_List_Nil);
var $author$project$Sharecrop$Types$RefreshReceived = function (a) {
	return {$: 'RefreshReceived', a: a};
};
var $author$project$Sharecrop$Generated$Auth$AuthResponse = F4(
	function (subjectKind, subjectID, accessToken, role) {
		return {accessToken: accessToken, role: role, subjectID: subjectID, subjectKind: subjectKind};
	});
var $elm$json$Json$Decode$map4 = _Json_map4;
var $elm$json$Json$Decode$string = _Json_decodeString;
var $author$project$Sharecrop$Generated$Auth$SubjectKindGuest = {$: 'SubjectKindGuest'};
var $author$project$Sharecrop$Generated$Auth$SubjectKindUser = {$: 'SubjectKindUser'};
var $elm$json$Json$Decode$fail = _Json_fail;
var $author$project$Sharecrop$Generated$Auth$subjectKindDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'user':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Auth$SubjectKindUser);
			case 'guest':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Auth$SubjectKindGuest);
			default:
				return $elm$json$Json$Decode$fail('invalid SubjectKind');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Auth$authResponseDecoder = A5(
	$elm$json$Json$Decode$map4,
	$author$project$Sharecrop$Generated$Auth$AuthResponse,
	A2($elm$json$Json$Decode$field, 'subject_kind', $author$project$Sharecrop$Generated$Auth$subjectKindDecoder),
	A2($elm$json$Json$Decode$field, 'subject_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'access_token', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'role', $elm$json$Json$Decode$string));
var $elm$http$Http$BadStatus_ = F2(
	function (a, b) {
		return {$: 'BadStatus_', a: a, b: b};
	});
var $elm$http$Http$BadUrl_ = function (a) {
	return {$: 'BadUrl_', a: a};
};
var $elm$http$Http$GoodStatus_ = F2(
	function (a, b) {
		return {$: 'GoodStatus_', a: a, b: b};
	});
var $elm$http$Http$NetworkError_ = {$: 'NetworkError_'};
var $elm$http$Http$Receiving = function (a) {
	return {$: 'Receiving', a: a};
};
var $elm$http$Http$Sending = function (a) {
	return {$: 'Sending', a: a};
};
var $elm$http$Http$Timeout_ = {$: 'Timeout_'};
var $elm$core$Dict$RBEmpty_elm_builtin = {$: 'RBEmpty_elm_builtin'};
var $elm$core$Dict$empty = $elm$core$Dict$RBEmpty_elm_builtin;
var $elm$core$Maybe$isJust = function (maybe) {
	if (maybe.$ === 'Just') {
		return true;
	} else {
		return false;
	}
};
var $elm$core$Platform$sendToSelf = _Platform_sendToSelf;
var $elm$core$Basics$compare = _Utils_compare;
var $elm$core$Dict$get = F2(
	function (targetKey, dict) {
		get:
		while (true) {
			if (dict.$ === 'RBEmpty_elm_builtin') {
				return $elm$core$Maybe$Nothing;
			} else {
				var key = dict.b;
				var value = dict.c;
				var left = dict.d;
				var right = dict.e;
				var _v1 = A2($elm$core$Basics$compare, targetKey, key);
				switch (_v1.$) {
					case 'LT':
						var $temp$targetKey = targetKey,
							$temp$dict = left;
						targetKey = $temp$targetKey;
						dict = $temp$dict;
						continue get;
					case 'EQ':
						return $elm$core$Maybe$Just(value);
					default:
						var $temp$targetKey = targetKey,
							$temp$dict = right;
						targetKey = $temp$targetKey;
						dict = $temp$dict;
						continue get;
				}
			}
		}
	});
var $elm$core$Dict$Black = {$: 'Black'};
var $elm$core$Dict$RBNode_elm_builtin = F5(
	function (a, b, c, d, e) {
		return {$: 'RBNode_elm_builtin', a: a, b: b, c: c, d: d, e: e};
	});
var $elm$core$Dict$Red = {$: 'Red'};
var $elm$core$Dict$balance = F5(
	function (color, key, value, left, right) {
		if ((right.$ === 'RBNode_elm_builtin') && (right.a.$ === 'Red')) {
			var _v1 = right.a;
			var rK = right.b;
			var rV = right.c;
			var rLeft = right.d;
			var rRight = right.e;
			if ((left.$ === 'RBNode_elm_builtin') && (left.a.$ === 'Red')) {
				var _v3 = left.a;
				var lK = left.b;
				var lV = left.c;
				var lLeft = left.d;
				var lRight = left.e;
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Red,
					key,
					value,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, lK, lV, lLeft, lRight),
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, rK, rV, rLeft, rRight));
			} else {
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					color,
					rK,
					rV,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, key, value, left, rLeft),
					rRight);
			}
		} else {
			if ((((left.$ === 'RBNode_elm_builtin') && (left.a.$ === 'Red')) && (left.d.$ === 'RBNode_elm_builtin')) && (left.d.a.$ === 'Red')) {
				var _v5 = left.a;
				var lK = left.b;
				var lV = left.c;
				var _v6 = left.d;
				var _v7 = _v6.a;
				var llK = _v6.b;
				var llV = _v6.c;
				var llLeft = _v6.d;
				var llRight = _v6.e;
				var lRight = left.e;
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Red,
					lK,
					lV,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, llK, llV, llLeft, llRight),
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, key, value, lRight, right));
			} else {
				return A5($elm$core$Dict$RBNode_elm_builtin, color, key, value, left, right);
			}
		}
	});
var $elm$core$Dict$insertHelp = F3(
	function (key, value, dict) {
		if (dict.$ === 'RBEmpty_elm_builtin') {
			return A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, key, value, $elm$core$Dict$RBEmpty_elm_builtin, $elm$core$Dict$RBEmpty_elm_builtin);
		} else {
			var nColor = dict.a;
			var nKey = dict.b;
			var nValue = dict.c;
			var nLeft = dict.d;
			var nRight = dict.e;
			var _v1 = A2($elm$core$Basics$compare, key, nKey);
			switch (_v1.$) {
				case 'LT':
					return A5(
						$elm$core$Dict$balance,
						nColor,
						nKey,
						nValue,
						A3($elm$core$Dict$insertHelp, key, value, nLeft),
						nRight);
				case 'EQ':
					return A5($elm$core$Dict$RBNode_elm_builtin, nColor, nKey, value, nLeft, nRight);
				default:
					return A5(
						$elm$core$Dict$balance,
						nColor,
						nKey,
						nValue,
						nLeft,
						A3($elm$core$Dict$insertHelp, key, value, nRight));
			}
		}
	});
var $elm$core$Dict$insert = F3(
	function (key, value, dict) {
		var _v0 = A3($elm$core$Dict$insertHelp, key, value, dict);
		if ((_v0.$ === 'RBNode_elm_builtin') && (_v0.a.$ === 'Red')) {
			var _v1 = _v0.a;
			var k = _v0.b;
			var v = _v0.c;
			var l = _v0.d;
			var r = _v0.e;
			return A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, k, v, l, r);
		} else {
			var x = _v0;
			return x;
		}
	});
var $elm$core$Dict$getMin = function (dict) {
	getMin:
	while (true) {
		if ((dict.$ === 'RBNode_elm_builtin') && (dict.d.$ === 'RBNode_elm_builtin')) {
			var left = dict.d;
			var $temp$dict = left;
			dict = $temp$dict;
			continue getMin;
		} else {
			return dict;
		}
	}
};
var $elm$core$Dict$moveRedLeft = function (dict) {
	if (((dict.$ === 'RBNode_elm_builtin') && (dict.d.$ === 'RBNode_elm_builtin')) && (dict.e.$ === 'RBNode_elm_builtin')) {
		if ((dict.e.d.$ === 'RBNode_elm_builtin') && (dict.e.d.a.$ === 'Red')) {
			var clr = dict.a;
			var k = dict.b;
			var v = dict.c;
			var _v1 = dict.d;
			var lClr = _v1.a;
			var lK = _v1.b;
			var lV = _v1.c;
			var lLeft = _v1.d;
			var lRight = _v1.e;
			var _v2 = dict.e;
			var rClr = _v2.a;
			var rK = _v2.b;
			var rV = _v2.c;
			var rLeft = _v2.d;
			var _v3 = rLeft.a;
			var rlK = rLeft.b;
			var rlV = rLeft.c;
			var rlL = rLeft.d;
			var rlR = rLeft.e;
			var rRight = _v2.e;
			return A5(
				$elm$core$Dict$RBNode_elm_builtin,
				$elm$core$Dict$Red,
				rlK,
				rlV,
				A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Black,
					k,
					v,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, lK, lV, lLeft, lRight),
					rlL),
				A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, rK, rV, rlR, rRight));
		} else {
			var clr = dict.a;
			var k = dict.b;
			var v = dict.c;
			var _v4 = dict.d;
			var lClr = _v4.a;
			var lK = _v4.b;
			var lV = _v4.c;
			var lLeft = _v4.d;
			var lRight = _v4.e;
			var _v5 = dict.e;
			var rClr = _v5.a;
			var rK = _v5.b;
			var rV = _v5.c;
			var rLeft = _v5.d;
			var rRight = _v5.e;
			if (clr.$ === 'Black') {
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Black,
					k,
					v,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, lK, lV, lLeft, lRight),
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, rK, rV, rLeft, rRight));
			} else {
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Black,
					k,
					v,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, lK, lV, lLeft, lRight),
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, rK, rV, rLeft, rRight));
			}
		}
	} else {
		return dict;
	}
};
var $elm$core$Dict$moveRedRight = function (dict) {
	if (((dict.$ === 'RBNode_elm_builtin') && (dict.d.$ === 'RBNode_elm_builtin')) && (dict.e.$ === 'RBNode_elm_builtin')) {
		if ((dict.d.d.$ === 'RBNode_elm_builtin') && (dict.d.d.a.$ === 'Red')) {
			var clr = dict.a;
			var k = dict.b;
			var v = dict.c;
			var _v1 = dict.d;
			var lClr = _v1.a;
			var lK = _v1.b;
			var lV = _v1.c;
			var _v2 = _v1.d;
			var _v3 = _v2.a;
			var llK = _v2.b;
			var llV = _v2.c;
			var llLeft = _v2.d;
			var llRight = _v2.e;
			var lRight = _v1.e;
			var _v4 = dict.e;
			var rClr = _v4.a;
			var rK = _v4.b;
			var rV = _v4.c;
			var rLeft = _v4.d;
			var rRight = _v4.e;
			return A5(
				$elm$core$Dict$RBNode_elm_builtin,
				$elm$core$Dict$Red,
				lK,
				lV,
				A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, llK, llV, llLeft, llRight),
				A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Black,
					k,
					v,
					lRight,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, rK, rV, rLeft, rRight)));
		} else {
			var clr = dict.a;
			var k = dict.b;
			var v = dict.c;
			var _v5 = dict.d;
			var lClr = _v5.a;
			var lK = _v5.b;
			var lV = _v5.c;
			var lLeft = _v5.d;
			var lRight = _v5.e;
			var _v6 = dict.e;
			var rClr = _v6.a;
			var rK = _v6.b;
			var rV = _v6.c;
			var rLeft = _v6.d;
			var rRight = _v6.e;
			if (clr.$ === 'Black') {
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Black,
					k,
					v,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, lK, lV, lLeft, lRight),
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, rK, rV, rLeft, rRight));
			} else {
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					$elm$core$Dict$Black,
					k,
					v,
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, lK, lV, lLeft, lRight),
					A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, rK, rV, rLeft, rRight));
			}
		}
	} else {
		return dict;
	}
};
var $elm$core$Dict$removeHelpPrepEQGT = F7(
	function (targetKey, dict, color, key, value, left, right) {
		if ((left.$ === 'RBNode_elm_builtin') && (left.a.$ === 'Red')) {
			var _v1 = left.a;
			var lK = left.b;
			var lV = left.c;
			var lLeft = left.d;
			var lRight = left.e;
			return A5(
				$elm$core$Dict$RBNode_elm_builtin,
				color,
				lK,
				lV,
				lLeft,
				A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Red, key, value, lRight, right));
		} else {
			_v2$2:
			while (true) {
				if ((right.$ === 'RBNode_elm_builtin') && (right.a.$ === 'Black')) {
					if (right.d.$ === 'RBNode_elm_builtin') {
						if (right.d.a.$ === 'Black') {
							var _v3 = right.a;
							var _v4 = right.d;
							var _v5 = _v4.a;
							return $elm$core$Dict$moveRedRight(dict);
						} else {
							break _v2$2;
						}
					} else {
						var _v6 = right.a;
						var _v7 = right.d;
						return $elm$core$Dict$moveRedRight(dict);
					}
				} else {
					break _v2$2;
				}
			}
			return dict;
		}
	});
var $elm$core$Dict$removeMin = function (dict) {
	if ((dict.$ === 'RBNode_elm_builtin') && (dict.d.$ === 'RBNode_elm_builtin')) {
		var color = dict.a;
		var key = dict.b;
		var value = dict.c;
		var left = dict.d;
		var lColor = left.a;
		var lLeft = left.d;
		var right = dict.e;
		if (lColor.$ === 'Black') {
			if ((lLeft.$ === 'RBNode_elm_builtin') && (lLeft.a.$ === 'Red')) {
				var _v3 = lLeft.a;
				return A5(
					$elm$core$Dict$RBNode_elm_builtin,
					color,
					key,
					value,
					$elm$core$Dict$removeMin(left),
					right);
			} else {
				var _v4 = $elm$core$Dict$moveRedLeft(dict);
				if (_v4.$ === 'RBNode_elm_builtin') {
					var nColor = _v4.a;
					var nKey = _v4.b;
					var nValue = _v4.c;
					var nLeft = _v4.d;
					var nRight = _v4.e;
					return A5(
						$elm$core$Dict$balance,
						nColor,
						nKey,
						nValue,
						$elm$core$Dict$removeMin(nLeft),
						nRight);
				} else {
					return $elm$core$Dict$RBEmpty_elm_builtin;
				}
			}
		} else {
			return A5(
				$elm$core$Dict$RBNode_elm_builtin,
				color,
				key,
				value,
				$elm$core$Dict$removeMin(left),
				right);
		}
	} else {
		return $elm$core$Dict$RBEmpty_elm_builtin;
	}
};
var $elm$core$Dict$removeHelp = F2(
	function (targetKey, dict) {
		if (dict.$ === 'RBEmpty_elm_builtin') {
			return $elm$core$Dict$RBEmpty_elm_builtin;
		} else {
			var color = dict.a;
			var key = dict.b;
			var value = dict.c;
			var left = dict.d;
			var right = dict.e;
			if (_Utils_cmp(targetKey, key) < 0) {
				if ((left.$ === 'RBNode_elm_builtin') && (left.a.$ === 'Black')) {
					var _v4 = left.a;
					var lLeft = left.d;
					if ((lLeft.$ === 'RBNode_elm_builtin') && (lLeft.a.$ === 'Red')) {
						var _v6 = lLeft.a;
						return A5(
							$elm$core$Dict$RBNode_elm_builtin,
							color,
							key,
							value,
							A2($elm$core$Dict$removeHelp, targetKey, left),
							right);
					} else {
						var _v7 = $elm$core$Dict$moveRedLeft(dict);
						if (_v7.$ === 'RBNode_elm_builtin') {
							var nColor = _v7.a;
							var nKey = _v7.b;
							var nValue = _v7.c;
							var nLeft = _v7.d;
							var nRight = _v7.e;
							return A5(
								$elm$core$Dict$balance,
								nColor,
								nKey,
								nValue,
								A2($elm$core$Dict$removeHelp, targetKey, nLeft),
								nRight);
						} else {
							return $elm$core$Dict$RBEmpty_elm_builtin;
						}
					}
				} else {
					return A5(
						$elm$core$Dict$RBNode_elm_builtin,
						color,
						key,
						value,
						A2($elm$core$Dict$removeHelp, targetKey, left),
						right);
				}
			} else {
				return A2(
					$elm$core$Dict$removeHelpEQGT,
					targetKey,
					A7($elm$core$Dict$removeHelpPrepEQGT, targetKey, dict, color, key, value, left, right));
			}
		}
	});
var $elm$core$Dict$removeHelpEQGT = F2(
	function (targetKey, dict) {
		if (dict.$ === 'RBNode_elm_builtin') {
			var color = dict.a;
			var key = dict.b;
			var value = dict.c;
			var left = dict.d;
			var right = dict.e;
			if (_Utils_eq(targetKey, key)) {
				var _v1 = $elm$core$Dict$getMin(right);
				if (_v1.$ === 'RBNode_elm_builtin') {
					var minKey = _v1.b;
					var minValue = _v1.c;
					return A5(
						$elm$core$Dict$balance,
						color,
						minKey,
						minValue,
						left,
						$elm$core$Dict$removeMin(right));
				} else {
					return $elm$core$Dict$RBEmpty_elm_builtin;
				}
			} else {
				return A5(
					$elm$core$Dict$balance,
					color,
					key,
					value,
					left,
					A2($elm$core$Dict$removeHelp, targetKey, right));
			}
		} else {
			return $elm$core$Dict$RBEmpty_elm_builtin;
		}
	});
var $elm$core$Dict$remove = F2(
	function (key, dict) {
		var _v0 = A2($elm$core$Dict$removeHelp, key, dict);
		if ((_v0.$ === 'RBNode_elm_builtin') && (_v0.a.$ === 'Red')) {
			var _v1 = _v0.a;
			var k = _v0.b;
			var v = _v0.c;
			var l = _v0.d;
			var r = _v0.e;
			return A5($elm$core$Dict$RBNode_elm_builtin, $elm$core$Dict$Black, k, v, l, r);
		} else {
			var x = _v0;
			return x;
		}
	});
var $elm$core$Dict$update = F3(
	function (targetKey, alter, dictionary) {
		var _v0 = alter(
			A2($elm$core$Dict$get, targetKey, dictionary));
		if (_v0.$ === 'Just') {
			var value = _v0.a;
			return A3($elm$core$Dict$insert, targetKey, value, dictionary);
		} else {
			return A2($elm$core$Dict$remove, targetKey, dictionary);
		}
	});
var $elm$http$Http$emptyBody = _Http_emptyBody;
var $elm$json$Json$Decode$decodeString = _Json_runOnString;
var $elm$core$Basics$composeR = F3(
	function (f, g, x) {
		return g(
			f(x));
	});
var $elm$http$Http$expectStringResponse = F2(
	function (toMsg, toResult) {
		return A3(
			_Http_expect,
			'',
			$elm$core$Basics$identity,
			A2($elm$core$Basics$composeR, toResult, toMsg));
	});
var $elm$core$Result$mapError = F2(
	function (f, result) {
		if (result.$ === 'Ok') {
			var v = result.a;
			return $elm$core$Result$Ok(v);
		} else {
			var e = result.a;
			return $elm$core$Result$Err(
				f(e));
		}
	});
var $elm$http$Http$BadBody = function (a) {
	return {$: 'BadBody', a: a};
};
var $elm$http$Http$BadStatus = function (a) {
	return {$: 'BadStatus', a: a};
};
var $elm$http$Http$BadUrl = function (a) {
	return {$: 'BadUrl', a: a};
};
var $elm$http$Http$NetworkError = {$: 'NetworkError'};
var $elm$http$Http$Timeout = {$: 'Timeout'};
var $elm$http$Http$resolve = F2(
	function (toResult, response) {
		switch (response.$) {
			case 'BadUrl_':
				var url = response.a;
				return $elm$core$Result$Err(
					$elm$http$Http$BadUrl(url));
			case 'Timeout_':
				return $elm$core$Result$Err($elm$http$Http$Timeout);
			case 'NetworkError_':
				return $elm$core$Result$Err($elm$http$Http$NetworkError);
			case 'BadStatus_':
				var metadata = response.a;
				return $elm$core$Result$Err(
					$elm$http$Http$BadStatus(metadata.statusCode));
			default:
				var body = response.b;
				return A2(
					$elm$core$Result$mapError,
					$elm$http$Http$BadBody,
					toResult(body));
		}
	});
var $elm$http$Http$expectJson = F2(
	function (toMsg, decoder) {
		return A2(
			$elm$http$Http$expectStringResponse,
			toMsg,
			$elm$http$Http$resolve(
				function (string) {
					return A2(
						$elm$core$Result$mapError,
						$elm$json$Json$Decode$errorToString,
						A2($elm$json$Json$Decode$decodeString, decoder, string));
				}));
	});
var $elm$http$Http$Request = function (a) {
	return {$: 'Request', a: a};
};
var $elm$http$Http$State = F2(
	function (reqs, subs) {
		return {reqs: reqs, subs: subs};
	});
var $elm$http$Http$init = $elm$core$Task$succeed(
	A2($elm$http$Http$State, $elm$core$Dict$empty, _List_Nil));
var $elm$core$Process$kill = _Scheduler_kill;
var $elm$core$Process$spawn = _Scheduler_spawn;
var $elm$http$Http$updateReqs = F3(
	function (router, cmds, reqs) {
		updateReqs:
		while (true) {
			if (!cmds.b) {
				return $elm$core$Task$succeed(reqs);
			} else {
				var cmd = cmds.a;
				var otherCmds = cmds.b;
				if (cmd.$ === 'Cancel') {
					var tracker = cmd.a;
					var _v2 = A2($elm$core$Dict$get, tracker, reqs);
					if (_v2.$ === 'Nothing') {
						var $temp$router = router,
							$temp$cmds = otherCmds,
							$temp$reqs = reqs;
						router = $temp$router;
						cmds = $temp$cmds;
						reqs = $temp$reqs;
						continue updateReqs;
					} else {
						var pid = _v2.a;
						return A2(
							$elm$core$Task$andThen,
							function (_v3) {
								return A3(
									$elm$http$Http$updateReqs,
									router,
									otherCmds,
									A2($elm$core$Dict$remove, tracker, reqs));
							},
							$elm$core$Process$kill(pid));
					}
				} else {
					var req = cmd.a;
					return A2(
						$elm$core$Task$andThen,
						function (pid) {
							var _v4 = req.tracker;
							if (_v4.$ === 'Nothing') {
								return A3($elm$http$Http$updateReqs, router, otherCmds, reqs);
							} else {
								var tracker = _v4.a;
								return A3(
									$elm$http$Http$updateReqs,
									router,
									otherCmds,
									A3($elm$core$Dict$insert, tracker, pid, reqs));
							}
						},
						$elm$core$Process$spawn(
							A3(
								_Http_toTask,
								router,
								$elm$core$Platform$sendToApp(router),
								req)));
				}
			}
		}
	});
var $elm$http$Http$onEffects = F4(
	function (router, cmds, subs, state) {
		return A2(
			$elm$core$Task$andThen,
			function (reqs) {
				return $elm$core$Task$succeed(
					A2($elm$http$Http$State, reqs, subs));
			},
			A3($elm$http$Http$updateReqs, router, cmds, state.reqs));
	});
var $elm$core$List$maybeCons = F3(
	function (f, mx, xs) {
		var _v0 = f(mx);
		if (_v0.$ === 'Just') {
			var x = _v0.a;
			return A2($elm$core$List$cons, x, xs);
		} else {
			return xs;
		}
	});
var $elm$core$List$filterMap = F2(
	function (f, xs) {
		return A3(
			$elm$core$List$foldr,
			$elm$core$List$maybeCons(f),
			_List_Nil,
			xs);
	});
var $elm$http$Http$maybeSend = F4(
	function (router, desiredTracker, progress, _v0) {
		var actualTracker = _v0.a;
		var toMsg = _v0.b;
		return _Utils_eq(desiredTracker, actualTracker) ? $elm$core$Maybe$Just(
			A2(
				$elm$core$Platform$sendToApp,
				router,
				toMsg(progress))) : $elm$core$Maybe$Nothing;
	});
var $elm$http$Http$onSelfMsg = F3(
	function (router, _v0, state) {
		var tracker = _v0.a;
		var progress = _v0.b;
		return A2(
			$elm$core$Task$andThen,
			function (_v1) {
				return $elm$core$Task$succeed(state);
			},
			$elm$core$Task$sequence(
				A2(
					$elm$core$List$filterMap,
					A3($elm$http$Http$maybeSend, router, tracker, progress),
					state.subs)));
	});
var $elm$http$Http$Cancel = function (a) {
	return {$: 'Cancel', a: a};
};
var $elm$http$Http$cmdMap = F2(
	function (func, cmd) {
		if (cmd.$ === 'Cancel') {
			var tracker = cmd.a;
			return $elm$http$Http$Cancel(tracker);
		} else {
			var r = cmd.a;
			return $elm$http$Http$Request(
				{
					allowCookiesFromOtherDomains: r.allowCookiesFromOtherDomains,
					body: r.body,
					expect: A2(_Http_mapExpect, func, r.expect),
					headers: r.headers,
					method: r.method,
					timeout: r.timeout,
					tracker: r.tracker,
					url: r.url
				});
		}
	});
var $elm$http$Http$MySub = F2(
	function (a, b) {
		return {$: 'MySub', a: a, b: b};
	});
var $elm$http$Http$subMap = F2(
	function (func, _v0) {
		var tracker = _v0.a;
		var toMsg = _v0.b;
		return A2(
			$elm$http$Http$MySub,
			tracker,
			A2($elm$core$Basics$composeR, toMsg, func));
	});
_Platform_effectManagers['Http'] = _Platform_createManager($elm$http$Http$init, $elm$http$Http$onEffects, $elm$http$Http$onSelfMsg, $elm$http$Http$cmdMap, $elm$http$Http$subMap);
var $elm$http$Http$command = _Platform_leaf('Http');
var $elm$http$Http$subscription = _Platform_leaf('Http');
var $elm$http$Http$request = function (r) {
	return $elm$http$Http$command(
		$elm$http$Http$Request(
			{allowCookiesFromOtherDomains: false, body: r.body, expect: r.expect, headers: r.headers, method: r.method, timeout: r.timeout, tracker: r.tracker, url: r.url}));
};
var $elm$http$Http$post = function (r) {
	return $elm$http$Http$request(
		{body: r.body, expect: r.expect, headers: _List_Nil, method: 'POST', timeout: $elm$core$Maybe$Nothing, tracker: $elm$core$Maybe$Nothing, url: r.url});
};
var $author$project$Sharecrop$Api$postRefresh = $elm$http$Http$post(
	{
		body: $elm$http$Http$emptyBody,
		expect: A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$RefreshReceived, $author$project$Sharecrop$Generated$Auth$authResponseDecoder),
		url: '/api/auth/refresh'
	});
var $author$project$Sharecrop$Types$CreateAttachmentFileChosen = function (a) {
	return {$: 'CreateAttachmentFileChosen', a: a};
};
var $author$project$Sharecrop$Types$LoggedIn = function (a) {
	return {$: 'LoggedIn', a: a};
};
var $author$project$Sharecrop$Types$OrgMembersReceived = function (a) {
	return {$: 'OrgMembersReceived', a: a};
};
var $author$project$Sharecrop$Types$SubmitAttachmentFileChosen = function (a) {
	return {$: 'SubmitAttachmentFileChosen', a: a};
};
var $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen = {$: 'TaskParticipationPolicyOpen'};
var $elm$core$Platform$Cmd$batch = _Platform_batch;
var $elm$core$Platform$Cmd$none = $elm$core$Platform$Cmd$batch(_List_Nil);
var $author$project$Sharecrop$Types$ReviewActionReceived = F2(
	function (a, b) {
		return {$: 'ReviewActionReceived', a: a, b: b};
	});
var $elm$json$Json$Encode$int = _Json_wrap;
var $elm$core$String$trim = _String_trim;
var $author$project$Sharecrop$Api$intInputOrZero = function (raw) {
	return A2(
		$elm$core$Maybe$withDefault,
		0,
		$elm$core$String$toInt(
			$elm$core$String$trim(raw)));
};
var $elm$json$Json$Encode$object = function (pairs) {
	return _Json_wrap(
		A3(
			$elm$core$List$foldl,
			F2(
				function (_v0, obj) {
					var k = _v0.a;
					var v = _v0.b;
					return A3(_Json_addField, k, v, obj);
				}),
			_Json_emptyObject(_Utils_Tuple0),
			pairs));
};
var $elm$json$Json$Encode$string = _Json_wrap;
var $author$project$Sharecrop$Api$acceptRequestBody = F4(
	function (submissionId, payoutAmount, tipAmount, tipCollectibleId) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'idempotency_key',
					$elm$json$Json$Encode$string('ui-accept:' + submissionId)),
					_Utils_Tuple2(
					'payout_amount',
					$elm$json$Json$Encode$int(
						$author$project$Sharecrop$Api$intInputOrZero(payoutAmount))),
					_Utils_Tuple2(
					'tip_amount',
					$elm$json$Json$Encode$int(
						$author$project$Sharecrop$Api$intInputOrZero(tipAmount))),
					_Utils_Tuple2(
					'tip_collectible_id',
					$elm$json$Json$Encode$string(tipCollectibleId))
				]));
	});
var $elm$http$Http$Header = F2(
	function (a, b) {
		return {$: 'Header', a: a, b: b};
	});
var $elm$http$Http$header = $elm$http$Http$Header;
var $author$project$Sharecrop$Api$authorizedRequest = F5(
	function (method, token, url, body, expect) {
		return $elm$http$Http$request(
			{
				body: body,
				expect: expect,
				headers: _List_fromArray(
					[
						A2($elm$http$Http$header, 'Authorization', 'Bearer ' + token)
					]),
				method: method,
				timeout: $elm$core$Maybe$Nothing,
				tracker: $elm$core$Maybe$Nothing,
				url: url
			});
	});
var $elm$http$Http$expectBytesResponse = F2(
	function (toMsg, toResult) {
		return A3(
			_Http_expect,
			'arraybuffer',
			_Http_toDataView,
			A2($elm$core$Basics$composeR, toResult, toMsg));
	});
var $elm$http$Http$expectWhatever = function (toMsg) {
	return A2(
		$elm$http$Http$expectBytesResponse,
		toMsg,
		$elm$http$Http$resolve(
			function (_v0) {
				return $elm$core$Result$Ok(_Utils_Tuple0);
			}));
};
var $elm$http$Http$jsonBody = function (value) {
	return A2(
		_Http_pair,
		'application/json',
		A2($elm$json$Json$Encode$encode, 0, value));
};
var $author$project$Sharecrop$Api$postAccept = F6(
	function (token, taskId, submissionId, payoutAmount, tipAmount, tipCollectibleId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/submissions/' + (submissionId + '/accept'))),
			$elm$http$Http$jsonBody(
				A4($author$project$Sharecrop$Api$acceptRequestBody, submissionId, payoutAmount, tipAmount, tipCollectibleId)),
			$elm$http$Http$expectWhatever(
				$author$project$Sharecrop$Types$ReviewActionReceived(submissionId)));
	});
var $author$project$Sharecrop$Api$updateLoggedIn = F2(
	function (model, change) {
		var _v0 = model.session;
		if (_v0.$ === 'LoggedIn') {
			var state = _v0.a;
			return _Utils_update(
				model,
				{
					session: $author$project$Sharecrop$Types$LoggedIn(
						change(state))
				});
		} else {
			return model;
		}
	});
var $author$project$Sharecrop$Api$acceptCommand = F3(
	function (model, state, submissionId) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reviewMessage: $elm$core$Maybe$Nothing});
					}),
				A6($author$project$Sharecrop$Api$postAccept, state.accessToken, taskId, submissionId, state.reviewPartialCredit, state.reviewTip, state.reviewTipCollectibleId));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Sharecrop$Types$SeriesCommentReceived = function (a) {
	return {$: 'SeriesCommentReceived', a: a};
};
var $author$project$Sharecrop$Generated$TaskSeries$SeriesCommentResponse = F5(
	function (id, seriesID, authorUserID, body, createdAt) {
		return {authorUserID: authorUserID, body: body, createdAt: createdAt, id: id, seriesID: seriesID};
	});
var $elm$json$Json$Decode$map5 = _Json_map5;
var $author$project$Sharecrop$Generated$TaskSeries$seriesCommentResponseDecoder = A6(
	$elm$json$Json$Decode$map5,
	$author$project$Sharecrop$Generated$TaskSeries$SeriesCommentResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'series_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'author_user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'body', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$addSeriesCommentCommand = F3(
	function (model, state, seriesId) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.seriesCommentBody)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							seriesMessage: $elm$core$Maybe$Just('A comment is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{seriesMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'POST',
				state.accessToken,
				'/api/task-series/' + (seriesId + '/comments'),
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(
						_List_fromArray(
							[
								_Utils_Tuple2(
								'body',
								$elm$json$Json$Encode$string(
									$elm$core$String$trim(state.seriesCommentBody)))
							]))),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesCommentReceived, $author$project$Sharecrop$Generated$TaskSeries$seriesCommentResponseDecoder)));
	});
var $author$project$Sharecrop$Types$SeriesMutationReceived = function (a) {
	return {$: 'SeriesMutationReceived', a: a};
};
var $author$project$Sharecrop$Types$SeriesDetailData = F3(
	function (series, tasks, comments) {
		return {comments: comments, series: series, tasks: tasks};
	});
var $elm$json$Json$Decode$list = _Json_decodeList;
var $elm$json$Json$Decode$map3 = _Json_map3;
var $author$project$Sharecrop$Types$SeriesTaskEntry = F3(
	function (id, title, state) {
		return {id: id, state: state, title: title};
	});
var $author$project$Sharecrop$Api$seriesTaskEntryDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Types$SeriesTaskEntry,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'title', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'state', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$TaskSeries$TaskSeriesResponse = F6(
	function (id, ownerKind, title, description, state, createdBy) {
		return {createdBy: createdBy, description: description, id: id, ownerKind: ownerKind, state: state, title: title};
	});
var $elm$json$Json$Decode$map6 = _Json_map6;
var $author$project$Sharecrop$Generated$TaskSeries$taskSeriesResponseDecoder = A7(
	$elm$json$Json$Decode$map6,
	$author$project$Sharecrop$Generated$TaskSeries$TaskSeriesResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'owner_kind', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'title', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'description', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'state', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$seriesDetailDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Types$SeriesDetailData,
	A2($elm$json$Json$Decode$field, 'series', $author$project$Sharecrop$Generated$TaskSeries$taskSeriesResponseDecoder),
	A2(
		$elm$json$Json$Decode$field,
		'tasks',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Api$seriesTaskEntryDecoder)),
	A2(
		$elm$json$Json$Decode$field,
		'comments',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$TaskSeries$seriesCommentResponseDecoder)));
var $author$project$Sharecrop$Api$addSeriesTaskCommand = F3(
	function (model, state, seriesId) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.addSeriesTaskId)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							seriesMessage: $elm$core$Maybe$Just('A task ID is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{seriesMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'POST',
				state.accessToken,
				'/api/task-series/' + (seriesId + '/tasks'),
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(
						_List_fromArray(
							[
								_Utils_Tuple2(
								'task_id',
								$elm$json$Json$Encode$string(
									$elm$core$String$trim(state.addSeriesTaskId)))
							]))),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesMutationReceived, $author$project$Sharecrop$Api$seriesDetailDecoder)));
	});
var $author$project$Sharecrop$Types$SubmissionCommentAdded = function (a) {
	return {$: 'SubmissionCommentAdded', a: a};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionCommentResponse = F5(
	function (id, submissionID, authorUserID, body, createdAt) {
		return {authorUserID: authorUserID, body: body, createdAt: createdAt, id: id, submissionID: submissionID};
	});
var $author$project$Sharecrop$Generated$Submission$submissionCommentResponseDecoder = A6(
	$elm$json$Json$Decode$map5,
	$author$project$Sharecrop$Generated$Submission$SubmissionCommentResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'submission_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'author_user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'body', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$addSubmissionComment = F3(
	function (token, submissionId, body) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/submissions/' + (submissionId + '/comments'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'body',
							$elm$json$Json$Encode$string(body))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SubmissionCommentAdded, $author$project$Sharecrop$Generated$Submission$submissionCommentResponseDecoder));
	});
var $elm$core$List$filter = F2(
	function (isGood, list) {
		return A3(
			$elm$core$List$foldr,
			F2(
				function (x, xs) {
					return isGood(x) ? A2($elm$core$List$cons, x, xs) : xs;
				}),
			_List_Nil,
			list);
	});
var $elm$core$Basics$neq = _Utils_notEqual;
var $author$project$Sharecrop$View$enumValueList = function (raw) {
	return A2(
		$elm$core$List$filter,
		function (value) {
			return value !== '';
		},
		A2(
			$elm$core$List$map,
			$elm$core$String$trim,
			A2($elm$core$String$split, ',', raw)));
};
var $elm$json$Json$Encode$list = F2(
	function (func, entries) {
		return _Json_wrap(
			A3(
				$elm$core$List$foldl,
				_Json_addEntry(func),
				_Json_emptyArray(_Utils_Tuple0),
				entries));
	});
var $author$project$Sharecrop$View$encodeFieldSchema = function (field) {
	var _v0 = field.kind;
	switch (_v0) {
		case 'enum':
			return $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('enum')),
						_Utils_Tuple2(
						'values',
						A2(
							$elm$json$Json$Encode$list,
							$elm$json$Json$Encode$string,
							$author$project$Sharecrop$View$enumValueList(field.enumValues)))
					]));
		case 'array':
			return $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('array')),
						_Utils_Tuple2(
						'item',
						$elm$json$Json$Encode$object(
							_List_fromArray(
								[
									_Utils_Tuple2(
									'kind',
									$elm$json$Json$Encode$string(field.itemKind))
								])))
					]));
		default:
			var other = _v0;
			return $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string(other))
					]));
	}
};
var $author$project$Sharecrop$View$encodeSchemaField = function (field) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'name',
				$elm$json$Json$Encode$string(
					$elm$core$String$trim(field.name))),
				_Utils_Tuple2(
				'presence',
				$elm$json$Json$Encode$string(
					field.required ? 'required' : 'may_omit')),
				_Utils_Tuple2(
				'schema',
				$author$project$Sharecrop$View$encodeFieldSchema(field))
			]));
};
var $elm$core$List$isEmpty = function (xs) {
	if (!xs.b) {
		return true;
	} else {
		return false;
	}
};
var $author$project$Sharecrop$View$schemaFromFields = function (fields) {
	var named = A2(
		$elm$core$List$filter,
		function (field) {
			return $elm$core$String$trim(field.name) !== '';
		},
		fields);
	return $elm$core$List$isEmpty(named) ? '{\"kind\":\"freeform\"}' : A2(
		$elm$json$Json$Encode$encode,
		0,
		$elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'kind',
					$elm$json$Json$Encode$string('object')),
					_Utils_Tuple2(
					'fields',
					A2($elm$json$Json$Encode$list, $author$project$Sharecrop$View$encodeSchemaField, named))
				])));
};
var $author$project$Main$applySchemaFields = F2(
	function (transform, state) {
		var nextFields = transform(state.createSchemaFields);
		return _Utils_update(
			state,
			{
				createResponseSchema: $author$project$Sharecrop$View$schemaFromFields(nextFields),
				createSchemaFields: nextFields
			});
	});
var $author$project$Main$attachmentMaxCount = 5;
var $author$project$Sharecrop$Types$AwardReceived = function (a) {
	return {$: 'AwardReceived', a: a};
};
var $author$project$Sharecrop$Generated$Collectible$CollectibleResponse = F9(
	function (id, name, kind, state, transferPolicy, ownerID, ownerKind, organizationID, art) {
		return {art: art, id: id, kind: kind, name: name, organizationID: organizationID, ownerID: ownerID, ownerKind: ownerKind, state: state, transferPolicy: transferPolicy};
	});
var $author$project$Sharecrop$Generated$Collectible$CollectibleKindBadge = {$: 'CollectibleKindBadge'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleKindEdition = {$: 'CollectibleKindEdition'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleKindUnique = {$: 'CollectibleKindUnique'};
var $author$project$Sharecrop$Generated$Collectible$collectibleKindDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'unique':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleKindUnique);
			case 'edition':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleKindEdition);
			case 'badge':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleKindBadge);
			default:
				return $elm$json$Json$Decode$fail('invalid CollectibleKind');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Collectible$CollectibleStateAwarded = {$: 'CollectibleStateAwarded'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleStateEscrowed = {$: 'CollectibleStateEscrowed'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleStateMinted = {$: 'CollectibleStateMinted'};
var $author$project$Sharecrop$Generated$Collectible$collectibleStateDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'minted':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleStateMinted);
			case 'escrowed':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleStateEscrowed);
			case 'awarded':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleStateAwarded);
			default:
				return $elm$json$Json$Decode$fail('invalid CollectibleState');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyIssuerControlled = {$: 'CollectibleTransferPolicyIssuerControlled'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyNonTransferableExceptPayout = {$: 'CollectibleTransferPolicyNonTransferableExceptPayout'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableBetweenUsers = {$: 'CollectibleTransferPolicyTransferableBetweenUsers'};
var $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableWithinOrganization = {$: 'CollectibleTransferPolicyTransferableWithinOrganization'};
var $author$project$Sharecrop$Generated$Collectible$collectibleTransferPolicyDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'non_transferable_except_payout':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyNonTransferableExceptPayout);
			case 'transferable_between_users':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableBetweenUsers);
			case 'transferable_within_organization':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableWithinOrganization);
			case 'issuer_controlled':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyIssuerControlled);
			default:
				return $elm$json$Json$Decode$fail('invalid CollectibleTransferPolicy');
		}
	},
	$elm$json$Json$Decode$string);
var $elm$json$Json$Decode$map8 = _Json_map8;
var $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A2(
			$elm$json$Json$Decode$map,
			finish,
			A2($elm$json$Json$Decode$field, 'art', $elm$json$Json$Decode$string));
	},
	A9(
		$elm$json$Json$Decode$map8,
		$author$project$Sharecrop$Generated$Collectible$CollectibleResponse,
		A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'kind', $author$project$Sharecrop$Generated$Collectible$collectibleKindDecoder),
		A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Collectible$collectibleStateDecoder),
		A2($elm$json$Json$Decode$field, 'transfer_policy', $author$project$Sharecrop$Generated$Collectible$collectibleTransferPolicyDecoder),
		A2($elm$json$Json$Decode$field, 'owner_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'owner_kind', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'organization_id', $elm$json$Json$Decode$string)));
var $author$project$Sharecrop$Api$collectibleRewardRequestBody = function (collectibleId) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'collectible_id',
				$elm$json$Json$Encode$string(collectibleId))
			]));
};
var $author$project$Sharecrop$Api$postCollectibleReward = F3(
	function (token, taskId, collectibleId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/collectible-reward'),
			$elm$http$Http$jsonBody(
				$author$project$Sharecrop$Api$collectibleRewardRequestBody(collectibleId)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AwardReceived, $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder));
	});
var $author$project$Sharecrop$Api$awardCommand = F3(
	function (model, state, collectibleId) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.awardTaskId)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							awardMessage: $elm$core$Maybe$Just('Task ID is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{awardMessage: $elm$core$Maybe$Nothing});
				}),
			A3($author$project$Sharecrop$Api$postCollectibleReward, state.accessToken, state.awardTaskId, collectibleId));
	});
var $author$project$Sharecrop$Types$AwardDefaultReceived = function (a) {
	return {$: 'AwardDefaultReceived', a: a};
};
var $author$project$Sharecrop$Api$awardDefaultCollectible = F4(
	function (token, slug, recipientKind, recipientId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/collectibles/award',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'slug',
							$elm$json$Json$Encode$string(slug)),
							_Utils_Tuple2(
							'recipient_kind',
							$elm$json$Json$Encode$string(recipientKind)),
							_Utils_Tuple2(
							'recipient_id',
							$elm$json$Json$Encode$string(recipientId))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AwardDefaultReceived, $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder));
	});
var $author$project$Sharecrop$Labels$collectibleStateLabel = function (state) {
	switch (state.$) {
		case 'CollectibleStateMinted':
			return 'minted';
		case 'CollectibleStateEscrowed':
			return 'escrowed';
		default:
			return 'awarded';
	}
};
var $author$project$Sharecrop$View$awardSuccessLabel = function (collectible) {
	return 'Awarded ' + (collectible.name + (' (' + ($author$project$Sharecrop$Labels$collectibleStateLabel(collectible.state) + ').')));
};
var $author$project$Sharecrop$Api$balanceFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return $elm$core$Maybe$Just(response.amount);
	} else {
		return $elm$core$Maybe$Nothing;
	}
};
var $author$project$Sharecrop$Types$AccountActionReceived = function (a) {
	return {$: 'AccountActionReceived', a: a};
};
var $author$project$Sharecrop$Api$changePassword = F3(
	function (token, current, next) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'PATCH',
			token,
			'/api/account/password',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'current_password',
							$elm$json$Json$Encode$string(current)),
							_Utils_Tuple2(
							'new_password',
							$elm$json$Json$Encode$string(next))
						]))),
			$elm$http$Http$expectWhatever($author$project$Sharecrop$Types$AccountActionReceived));
	});
var $author$project$Sharecrop$Api$collectiblesFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.collectibles;
	} else {
		return _List_Nil;
	}
};
var $author$project$Sharecrop$Api$confirmEmailVerification = F2(
	function (token, accountToken) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/auth/email-verification/confirm',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'token',
							$elm$json$Json$Encode$string(accountToken))
						]))),
			$elm$http$Http$expectWhatever($author$project$Sharecrop$Types$AccountActionReceived));
	});
var $author$project$Sharecrop$Types$PasswordResetConfirmed = function (a) {
	return {$: 'PasswordResetConfirmed', a: a};
};
var $author$project$Sharecrop$Api$confirmPasswordReset = function (model) {
	return $elm$http$Http$post(
		{
			body: $elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'token',
							$elm$json$Json$Encode$string(model.resetToken)),
							_Utils_Tuple2(
							'password',
							$elm$json$Json$Encode$string(model.resetPassword))
						]))),
			expect: $elm$http$Http$expectWhatever($author$project$Sharecrop$Types$PasswordResetConfirmed),
			url: '/api/auth/password-reset/confirm'
		});
};
var $author$project$Main$copyToClipboard = _Platform_outgoingPort('copyToClipboard', $elm$json$Json$Encode$string);
var $author$project$Sharecrop$Types$AgentCreated = function (a) {
	return {$: 'AgentCreated', a: a};
};
var $author$project$Sharecrop$Generated$Agent$AgentCredentialCreatedResponse = F2(
	function (credential, secret) {
		return {credential: credential, secret: secret};
	});
var $author$project$Sharecrop$Generated$Agent$AgentCredentialResponse = F4(
	function (id, label, scopes, state) {
		return {id: id, label: label, scopes: scopes, state: state};
	});
var $author$project$Sharecrop$Generated$Agent$AgentCredentialStateActive = {$: 'AgentCredentialStateActive'};
var $author$project$Sharecrop$Generated$Agent$AgentCredentialStateRevoked = {$: 'AgentCredentialStateRevoked'};
var $author$project$Sharecrop$Generated$Agent$agentCredentialStateDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'active':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentCredentialStateActive);
			case 'revoked':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentCredentialStateRevoked);
			default:
				return $elm$json$Json$Decode$fail('invalid AgentCredentialState');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsRead = {$: 'AgentScopeSubmissionsRead'};
var $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsReview = {$: 'AgentScopeSubmissionsReview'};
var $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite = {$: 'AgentScopeSubmissionsWrite'};
var $author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead = {$: 'AgentScopeTasksRead'};
var $author$project$Sharecrop$Generated$Agent$AgentScopeTasksWrite = {$: 'AgentScopeTasksWrite'};
var $author$project$Sharecrop$Generated$Agent$agentScopeDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'tasks_read':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead);
			case 'tasks_write':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentScopeTasksWrite);
			case 'submissions_write':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite);
			case 'submissions_read':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsRead);
			case 'submissions_review':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsReview);
			default:
				return $elm$json$Json$Decode$fail('invalid AgentScope');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Agent$agentCredentialResponseDecoder = A5(
	$elm$json$Json$Decode$map4,
	$author$project$Sharecrop$Generated$Agent$AgentCredentialResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'label', $elm$json$Json$Decode$string),
	A2(
		$elm$json$Json$Decode$field,
		'scopes',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Agent$agentScopeDecoder)),
	A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Agent$agentCredentialStateDecoder));
var $author$project$Sharecrop$Generated$Agent$agentCredentialCreatedResponseDecoder = A3(
	$elm$json$Json$Decode$map2,
	$author$project$Sharecrop$Generated$Agent$AgentCredentialCreatedResponse,
	A2($elm$json$Json$Decode$field, 'credential', $author$project$Sharecrop$Generated$Agent$agentCredentialResponseDecoder),
	A2($elm$json$Json$Decode$field, 'secret', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Agent$agentScopeEncoder = function (agentScope) {
	switch (agentScope.$) {
		case 'AgentScopeTasksRead':
			return $elm$json$Json$Encode$string('tasks_read');
		case 'AgentScopeTasksWrite':
			return $elm$json$Json$Encode$string('tasks_write');
		case 'AgentScopeSubmissionsWrite':
			return $elm$json$Json$Encode$string('submissions_write');
		case 'AgentScopeSubmissionsRead':
			return $elm$json$Json$Encode$string('submissions_read');
		default:
			return $elm$json$Json$Encode$string('submissions_review');
	}
};
var $author$project$Sharecrop$Api$agentRequestBody = F2(
	function (agentLabel, scopes) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'label',
					$elm$json$Json$Encode$string(agentLabel)),
					_Utils_Tuple2(
					'scopes',
					A2($elm$json$Json$Encode$list, $author$project$Sharecrop$Generated$Agent$agentScopeEncoder, scopes))
				]));
	});
var $author$project$Sharecrop$Api$postAgent = F3(
	function (token, agentLabel, scopes) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/agent-credentials',
			$elm$http$Http$jsonBody(
				A2($author$project$Sharecrop$Api$agentRequestBody, agentLabel, scopes)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AgentCreated, $author$project$Sharecrop$Generated$Agent$agentCredentialCreatedResponseDecoder));
	});
var $author$project$Sharecrop$Api$createAgentCommand = F2(
	function (model, state) {
		return $elm$core$List$isEmpty(state.agentScopes) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							agentMessage: $elm$core$Maybe$Just('Select at least one scope.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{agentMessage: $elm$core$Maybe$Nothing, newCredential: $elm$core$Maybe$Nothing});
				}),
			A3($author$project$Sharecrop$Api$postAgent, state.accessToken, state.agentLabel, state.agentScopes));
	});
var $author$project$Sharecrop$Types$CreateOrgReceived = function (a) {
	return {$: 'CreateOrgReceived', a: a};
};
var $author$project$Sharecrop$Generated$Organization$OrganizationResponse = F3(
	function (id, name, createdBy) {
		return {createdBy: createdBy, id: id, name: name};
	});
var $author$project$Sharecrop$Generated$Organization$organizationResponseDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Generated$Organization$OrganizationResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$createOrgCommand = F2(
	function (model, state) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.createOrgName)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							orgMessage: $elm$core$Maybe$Just('Organization name is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{orgMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'POST',
				state.accessToken,
				'/api/organizations',
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(
						_List_fromArray(
							[
								_Utils_Tuple2(
								'name',
								$elm$json$Json$Encode$string(
									$elm$core$String$trim(state.createOrgName)))
							]))),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CreateOrgReceived, $author$project$Sharecrop$Generated$Organization$organizationResponseDecoder)));
	});
var $author$project$Sharecrop$Types$CreateOrgTeamReceived = function (a) {
	return {$: 'CreateOrgTeamReceived', a: a};
};
var $author$project$Sharecrop$Generated$Team$TeamResponse = F6(
	function (id, ownerKind, organizationID, ownerUserID, name, createdBy) {
		return {createdBy: createdBy, id: id, name: name, organizationID: organizationID, ownerKind: ownerKind, ownerUserID: ownerUserID};
	});
var $author$project$Sharecrop$Generated$Team$teamResponseDecoder = A7(
	$elm$json$Json$Decode$map6,
	$author$project$Sharecrop$Generated$Team$TeamResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'owner_kind', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'organization_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'owner_user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$createOrgTeamCommand = F2(
	function (model, state) {
		return ($elm$core$String$isEmpty(
			$elm$core$String$trim(state.createOrgTeamName)) || (state.activeOrgId === '')) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							orgTeamMessage: $elm$core$Maybe$Just('A team name is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{orgTeamMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'POST',
				state.accessToken,
				'/api/organizations/' + (state.activeOrgId + '/teams'),
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(
						_List_fromArray(
							[
								_Utils_Tuple2(
								'name',
								$elm$json$Json$Encode$string(
									$elm$core$String$trim(state.createOrgTeamName)))
							]))),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CreateOrgTeamReceived, $author$project$Sharecrop$Generated$Team$teamResponseDecoder)));
	});
var $author$project$Sharecrop$Api$seriesBody = F2(
	function (title, description) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'title',
					$elm$json$Json$Encode$string(
						$elm$core$String$trim(title))),
					_Utils_Tuple2(
					'description',
					$elm$json$Json$Encode$string(description))
				]));
	});
var $author$project$Sharecrop$Api$createSeriesCommand = F2(
	function (model, state) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.createSeriesTitle)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							seriesMessage: $elm$core$Maybe$Just('A series title is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{seriesMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'POST',
				state.accessToken,
				'/api/task-series',
				$elm$http$Http$jsonBody(
					A2($author$project$Sharecrop$Api$seriesBody, state.createSeriesTitle, state.createSeriesDescription)),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesMutationReceived, $author$project$Sharecrop$Api$seriesDetailDecoder)));
	});
var $author$project$Sharecrop$Labels$participationUsesReservation = function (tag) {
	return (tag === 'reservation_required') || (tag === 'approval_required');
};
var $author$project$Sharecrop$Types$CreateTaskReceived = function (a) {
	return {$: 'CreateTaskReceived', a: a};
};
var $author$project$Sharecrop$Api$attachmentRequestBody = function (attachment) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'name',
				$elm$json$Json$Encode$string(attachment.name)),
				_Utils_Tuple2(
				'content_type',
				$elm$json$Json$Encode$string(attachment.contentType)),
				_Utils_Tuple2(
				'data_url',
				$elm$json$Json$Encode$string(attachment.dataURL))
			]));
};
var $author$project$Sharecrop$Api$createOwnerBody = function (state) {
	return (state.createTaskOwner === '') ? $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'kind',
				$elm$json$Json$Encode$string('user')),
				_Utils_Tuple2(
				'user_id',
				$elm$json$Json$Encode$string(state.subjectId)),
				_Utils_Tuple2(
				'team_id',
				$elm$json$Json$Encode$string('')),
				_Utils_Tuple2(
				'organization_id',
				$elm$json$Json$Encode$string(''))
			])) : $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'kind',
				$elm$json$Json$Encode$string('organization')),
				_Utils_Tuple2(
				'user_id',
				$elm$json$Json$Encode$string('')),
				_Utils_Tuple2(
				'team_id',
				$elm$json$Json$Encode$string('')),
				_Utils_Tuple2(
				'organization_id',
				$elm$json$Json$Encode$string(state.createTaskOwner))
			]));
};
var $author$project$Sharecrop$Labels$assigneeScopeTag = function (scope) {
	switch (scope.$) {
		case 'TaskAssigneeScopeUser':
			return 'user';
		case 'TaskAssigneeScopeOrganizationTeam':
			return 'organization_team';
		default:
			return 'team';
	}
};
var $author$project$Sharecrop$Api$reservationHoursValue = function (raw) {
	var _v0 = $elm$core$String$toInt(raw);
	if (_v0.$ === 'Just') {
		var hours = _v0.a;
		return hours;
	} else {
		return 48;
	}
};
var $author$project$Sharecrop$Api$createParticipationBody = function (state) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'policy',
				$elm$json$Json$Encode$string(state.createParticipationPolicy)),
				_Utils_Tuple2(
				'assignee_scope',
				$elm$json$Json$Encode$string(
					$author$project$Sharecrop$Labels$assigneeScopeTag(state.createAssigneeScope))),
				_Utils_Tuple2(
				'reservation_expiry_hours',
				$elm$json$Json$Encode$int(
					$author$project$Sharecrop$Api$reservationHoursValue(state.createReservationHours)))
			]));
};
var $author$project$Sharecrop$Api$createPayloadBody = function (state) {
	return ($elm$core$String$trim(state.createPayloadJson) === '') ? $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'kind',
				$elm$json$Json$Encode$string('none')),
				_Utils_Tuple2(
				'json',
				$elm$json$Json$Encode$string(''))
			])) : $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'kind',
				$elm$json$Json$Encode$string('json')),
				_Utils_Tuple2(
				'json',
				$elm$json$Json$Encode$string(state.createPayloadJson))
			]));
};
var $author$project$Sharecrop$Api$createRewardBody = F3(
	function (kind, rawAmount, collectibleIds) {
		var _v0 = $elm$core$String$toInt(rawAmount);
		if (_v0.$ === 'Just') {
			var amount = _v0.a;
			return ((kind === 'credit') && (amount > 0)) ? $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('credit')),
						_Utils_Tuple2(
						'credit_amount',
						$elm$json$Json$Encode$int(amount)),
						_Utils_Tuple2(
						'collectible_ids',
						A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, _List_Nil))
					])) : ((kind === 'collectible') ? $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('collectible')),
						_Utils_Tuple2(
						'credit_amount',
						$elm$json$Json$Encode$int(0)),
						_Utils_Tuple2(
						'collectible_ids',
						A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, collectibleIds))
					])) : (((kind === 'bundle') && (amount > 0)) ? $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('bundle')),
						_Utils_Tuple2(
						'credit_amount',
						$elm$json$Json$Encode$int(amount)),
						_Utils_Tuple2(
						'collectible_ids',
						A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, collectibleIds))
					])) : $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('none')),
						_Utils_Tuple2(
						'credit_amount',
						$elm$json$Json$Encode$int(0)),
						_Utils_Tuple2(
						'collectible_ids',
						A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, _List_Nil))
					]))));
		} else {
			return (kind === 'collectible') ? $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('collectible')),
						_Utils_Tuple2(
						'credit_amount',
						$elm$json$Json$Encode$int(0)),
						_Utils_Tuple2(
						'collectible_ids',
						A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, collectibleIds))
					])) : $elm$json$Json$Encode$object(
				_List_fromArray(
					[
						_Utils_Tuple2(
						'kind',
						$elm$json$Json$Encode$string('none')),
						_Utils_Tuple2(
						'credit_amount',
						$elm$json$Json$Encode$int(0)),
						_Utils_Tuple2(
						'collectible_ids',
						A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, _List_Nil))
					]));
		}
	});
var $author$project$Sharecrop$Api$createSchemaString = function (state) {
	return ($elm$core$String$trim(state.createResponseSchema) === '') ? '{\"kind\":\"freeform\"}' : state.createResponseSchema;
};
var $author$project$Sharecrop$Types$visibilityOrganizationTag = 'organization';
var $author$project$Sharecrop$Types$visibilityTeamTag = 'team';
var $author$project$Sharecrop$Types$visibilityUserTag = 'user';
var $author$project$Sharecrop$Api$createVisibilityBody = function (state) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'kind',
				$elm$json$Json$Encode$string(state.createVisibility)),
				_Utils_Tuple2(
				'user_id',
				$elm$json$Json$Encode$string(
					_Utils_eq(state.createVisibility, $author$project$Sharecrop$Types$visibilityUserTag) ? state.createScopeUserId : '')),
				_Utils_Tuple2(
				'team_id',
				$elm$json$Json$Encode$string(
					_Utils_eq(state.createVisibility, $author$project$Sharecrop$Types$visibilityTeamTag) ? state.createScopeTeamId : '')),
				_Utils_Tuple2(
				'organization_id',
				$elm$json$Json$Encode$string(
					_Utils_eq(state.createVisibility, $author$project$Sharecrop$Types$visibilityOrganizationTag) ? state.createScopeOrganizationId : ''))
			]));
};
var $author$project$Sharecrop$Api$createTaskRequestBody = function (state) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'owner',
				$author$project$Sharecrop$Api$createOwnerBody(state)),
				_Utils_Tuple2(
				'title',
				$elm$json$Json$Encode$string(state.createTitle)),
				_Utils_Tuple2(
				'description',
				$elm$json$Json$Encode$string(state.createDescription)),
				_Utils_Tuple2(
				'reward',
				A3($author$project$Sharecrop$Api$createRewardBody, state.createRewardKind, state.createRewardAmount, state.createRewardCollectibleIds)),
				_Utils_Tuple2(
				'participation',
				$author$project$Sharecrop$Api$createParticipationBody(state)),
				_Utils_Tuple2(
				'visibility',
				$author$project$Sharecrop$Api$createVisibilityBody(state)),
				_Utils_Tuple2(
				'placement',
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'kind',
							$elm$json$Json$Encode$string('standalone')),
							_Utils_Tuple2(
							'series_id',
							$elm$json$Json$Encode$string('')),
							_Utils_Tuple2(
							'series_title',
							$elm$json$Json$Encode$string('')),
							_Utils_Tuple2(
							'series_position',
							$elm$json$Json$Encode$int(0))
						]))),
				_Utils_Tuple2(
				'response_schema_json',
				$elm$json$Json$Encode$string(
					$author$project$Sharecrop$Api$createSchemaString(state))),
				_Utils_Tuple2(
				'payload',
				$author$project$Sharecrop$Api$createPayloadBody(state)),
				_Utils_Tuple2(
				'task_type',
				$elm$json$Json$Encode$string(state.createTaskType)),
				_Utils_Tuple2(
				'reference_url',
				$elm$json$Json$Encode$string(state.createReferenceURL)),
				_Utils_Tuple2(
				'attachments',
				A2($elm$json$Json$Encode$list, $author$project$Sharecrop$Api$attachmentRequestBody, state.createAttachments))
			]));
};
var $author$project$Sharecrop$Api$taskDetailFromResponse = function (response) {
	return {assigneeScope: response.assigneeScope, attachments: response.attachments, availabilityKind: response.availabilityKind, createdBy: response.createdBy, description: response.description, id: response.id, participationPolicy: response.participationPolicy, payloadJson: response.payloadJSON, payloadKind: response.payloadKind, referenceURL: response.referenceURL, reservationExpiryHours: response.reservationExpiryHours, responseSchemaJson: response.responseSchemaJSON, reviewerAction: response.reviewerAction, rewardCollectibleCount: response.rewardCollectibleCount, rewardCreditAmount: response.rewardCreditAmount, rewardKind: response.rewardKind, seriesID: response.seriesID, state: response.state, taskType: response.taskType, title: response.title, viewerAction: response.viewerAction};
};
var $author$project$Sharecrop$Generated$Task$TaskResponse = function (id) {
	return function (ownerKind) {
		return function (ownerID) {
			return function (title) {
				return function (description) {
					return function (taskType) {
						return function (referenceURL) {
							return function (rewardKind) {
								return function (rewardCreditAmount) {
									return function (rewardCollectibleCount) {
										return function (participationPolicy) {
											return function (assigneeScope) {
												return function (reservationExpiryHours) {
													return function (state) {
														return function (visibilityKind) {
															return function (visibilityID) {
																return function (availabilityKind) {
																	return function (viewerAction) {
																		return function (reviewerAction) {
																			return function (seriesKind) {
																				return function (seriesID) {
																					return function (seriesPosition) {
																						return function (responseSchemaJSON) {
																							return function (payloadKind) {
																								return function (payloadJSON) {
																									return function (attachments) {
																										return function (createdBy) {
																											return {assigneeScope: assigneeScope, attachments: attachments, availabilityKind: availabilityKind, createdBy: createdBy, description: description, id: id, ownerID: ownerID, ownerKind: ownerKind, participationPolicy: participationPolicy, payloadJSON: payloadJSON, payloadKind: payloadKind, referenceURL: referenceURL, reservationExpiryHours: reservationExpiryHours, responseSchemaJSON: responseSchemaJSON, reviewerAction: reviewerAction, rewardCollectibleCount: rewardCollectibleCount, rewardCreditAmount: rewardCreditAmount, rewardKind: rewardKind, seriesID: seriesID, seriesKind: seriesKind, seriesPosition: seriesPosition, state: state, taskType: taskType, title: title, viewerAction: viewerAction, visibilityID: visibilityID, visibilityKind: visibilityKind};
																										};
																									};
																								};
																							};
																						};
																					};
																				};
																			};
																		};
																	};
																};
															};
														};
													};
												};
											};
										};
									};
								};
							};
						};
					};
				};
			};
		};
	};
};
var $elm$json$Json$Decode$int = _Json_decodeInt;
var $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeOrganizationTeam = {$: 'TaskAssigneeScopeOrganizationTeam'};
var $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeTeam = {$: 'TaskAssigneeScopeTeam'};
var $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser = {$: 'TaskAssigneeScopeUser'};
var $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'user':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser);
			case 'organization_team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAssigneeScopeOrganizationTeam);
			case 'team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAssigneeScopeTeam);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskAssigneeScope');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$TaskAttachmentResponse = F4(
	function (name, contentType, sizeBytes, dataURL) {
		return {contentType: contentType, dataURL: dataURL, name: name, sizeBytes: sizeBytes};
	});
var $author$project$Sharecrop$Generated$Task$taskAttachmentResponseDecoder = A5(
	$elm$json$Json$Decode$map4,
	$author$project$Sharecrop$Generated$Task$TaskAttachmentResponse,
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'content_type', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'size_bytes', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'data_url', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Task$TaskAvailabilityKindAvailable = {$: 'TaskAvailabilityKindAvailable'};
var $author$project$Sharecrop$Generated$Task$TaskAvailabilityKindAwaitingApproval = {$: 'TaskAvailabilityKindAwaitingApproval'};
var $author$project$Sharecrop$Generated$Task$TaskAvailabilityKindClosed = {$: 'TaskAvailabilityKindClosed'};
var $author$project$Sharecrop$Generated$Task$TaskAvailabilityKindReserved = {$: 'TaskAvailabilityKindReserved'};
var $author$project$Sharecrop$Generated$Task$taskAvailabilityKindDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'available':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAvailabilityKindAvailable);
			case 'reserved':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAvailabilityKindReserved);
			case 'awaiting_approval':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAvailabilityKindAwaitingApproval);
			case 'closed':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAvailabilityKindClosed);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskAvailabilityKind');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$TaskOwnerKindOrganization = {$: 'TaskOwnerKindOrganization'};
var $author$project$Sharecrop$Generated$Task$TaskOwnerKindOrganizationTeam = {$: 'TaskOwnerKindOrganizationTeam'};
var $author$project$Sharecrop$Generated$Task$TaskOwnerKindTeam = {$: 'TaskOwnerKindTeam'};
var $author$project$Sharecrop$Generated$Task$TaskOwnerKindUser = {$: 'TaskOwnerKindUser'};
var $author$project$Sharecrop$Generated$Task$taskOwnerKindDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'user':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskOwnerKindUser);
			case 'team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskOwnerKindTeam);
			case 'organization':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskOwnerKindOrganization);
			case 'organization_team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskOwnerKindOrganizationTeam);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskOwnerKind');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyApprovalRequired = {$: 'TaskParticipationPolicyApprovalRequired'};
var $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyReservationRequired = {$: 'TaskParticipationPolicyReservationRequired'};
var $author$project$Sharecrop$Generated$Task$taskParticipationPolicyDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'open':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen);
			case 'reservation_required':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyReservationRequired);
			case 'approval_required':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyApprovalRequired);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskParticipationPolicy');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$TaskStateCancelled = {$: 'TaskStateCancelled'};
var $author$project$Sharecrop$Generated$Task$TaskStateClosed = {$: 'TaskStateClosed'};
var $author$project$Sharecrop$Generated$Task$TaskStateDraft = {$: 'TaskStateDraft'};
var $author$project$Sharecrop$Generated$Task$TaskStateExpired = {$: 'TaskStateExpired'};
var $author$project$Sharecrop$Generated$Task$TaskStateOpen = {$: 'TaskStateOpen'};
var $author$project$Sharecrop$Generated$Task$taskStateDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'draft':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskStateDraft);
			case 'open':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskStateOpen);
			case 'closed':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskStateClosed);
			case 'cancelled':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskStateCancelled);
			case 'expired':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskStateExpired);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskState');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$TaskViewerActionNone = {$: 'TaskViewerActionNone'};
var $author$project$Sharecrop$Generated$Task$TaskViewerActionRequestApproval = {$: 'TaskViewerActionRequestApproval'};
var $author$project$Sharecrop$Generated$Task$TaskViewerActionReserve = {$: 'TaskViewerActionReserve'};
var $author$project$Sharecrop$Generated$Task$TaskViewerActionSubmit = {$: 'TaskViewerActionSubmit'};
var $author$project$Sharecrop$Generated$Task$TaskViewerActionWait = {$: 'TaskViewerActionWait'};
var $author$project$Sharecrop$Generated$Task$taskViewerActionDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'submit':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskViewerActionSubmit);
			case 'reserve':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskViewerActionReserve);
			case 'request_approval':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskViewerActionRequestApproval);
			case 'wait':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskViewerActionWait);
			case 'none':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskViewerActionNone);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskViewerAction');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$TaskVisibilityKindOrganization = {$: 'TaskVisibilityKindOrganization'};
var $author$project$Sharecrop$Generated$Task$TaskVisibilityKindOrganizationTeam = {$: 'TaskVisibilityKindOrganizationTeam'};
var $author$project$Sharecrop$Generated$Task$TaskVisibilityKindPublic = {$: 'TaskVisibilityKindPublic'};
var $author$project$Sharecrop$Generated$Task$TaskVisibilityKindTeam = {$: 'TaskVisibilityKindTeam'};
var $author$project$Sharecrop$Generated$Task$TaskVisibilityKindUser = {$: 'TaskVisibilityKindUser'};
var $author$project$Sharecrop$Generated$Task$taskVisibilityKindDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'public':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskVisibilityKindPublic);
			case 'user':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskVisibilityKindUser);
			case 'team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskVisibilityKindTeam);
			case 'organization':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskVisibilityKindOrganization);
			case 'organization_team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskVisibilityKindOrganizationTeam);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskVisibilityKind');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$taskResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A4(
			$elm$json$Json$Decode$map3,
			finish,
			A2($elm$json$Json$Decode$field, 'payload_json', $elm$json$Json$Decode$string),
			A2(
				$elm$json$Json$Decode$field,
				'attachments',
				$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Task$taskAttachmentResponseDecoder)),
			A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string));
	},
	A2(
		$elm$json$Json$Decode$andThen,
		function (finish) {
			return A9(
				$elm$json$Json$Decode$map8,
				finish,
				A2($elm$json$Json$Decode$field, 'availability_kind', $author$project$Sharecrop$Generated$Task$taskAvailabilityKindDecoder),
				A2($elm$json$Json$Decode$field, 'viewer_action', $author$project$Sharecrop$Generated$Task$taskViewerActionDecoder),
				A2($elm$json$Json$Decode$field, 'reviewer_action', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'series_kind', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'series_id', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'series_position', $elm$json$Json$Decode$int),
				A2($elm$json$Json$Decode$field, 'response_schema_json', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'payload_kind', $elm$json$Json$Decode$string));
		},
		A2(
			$elm$json$Json$Decode$andThen,
			function (finish) {
				return A9(
					$elm$json$Json$Decode$map8,
					finish,
					A2($elm$json$Json$Decode$field, 'reward_credit_amount', $elm$json$Json$Decode$int),
					A2($elm$json$Json$Decode$field, 'reward_collectible_count', $elm$json$Json$Decode$int),
					A2($elm$json$Json$Decode$field, 'participation_policy', $author$project$Sharecrop$Generated$Task$taskParticipationPolicyDecoder),
					A2($elm$json$Json$Decode$field, 'assignee_scope', $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder),
					A2($elm$json$Json$Decode$field, 'reservation_expiry_hours', $elm$json$Json$Decode$int),
					A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Task$taskStateDecoder),
					A2($elm$json$Json$Decode$field, 'visibility_kind', $author$project$Sharecrop$Generated$Task$taskVisibilityKindDecoder),
					A2($elm$json$Json$Decode$field, 'visibility_id', $elm$json$Json$Decode$string));
			},
			A9(
				$elm$json$Json$Decode$map8,
				$author$project$Sharecrop$Generated$Task$TaskResponse,
				A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'owner_kind', $author$project$Sharecrop$Generated$Task$taskOwnerKindDecoder),
				A2($elm$json$Json$Decode$field, 'owner_id', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'title', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'description', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'task_type', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'reference_url', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'reward_kind', $elm$json$Json$Decode$string)))));
var $author$project$Sharecrop$Api$taskDetailDecoder = A2($elm$json$Json$Decode$map, $author$project$Sharecrop$Api$taskDetailFromResponse, $author$project$Sharecrop$Generated$Task$taskResponseDecoder);
var $author$project$Sharecrop$Api$postCreateTask = function (state) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'POST',
		state.accessToken,
		'/api/tasks',
		$elm$http$Http$jsonBody(
			$author$project$Sharecrop$Api$createTaskRequestBody(state)),
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CreateTaskReceived, $author$project$Sharecrop$Api$taskDetailDecoder));
};
var $author$project$Sharecrop$Api$createTaskCommand = F2(
	function (model, state) {
		return ($elm$core$String$isEmpty(
			$elm$core$String$trim(state.createTitle)) || $elm$core$String$isEmpty(
			$elm$core$String$trim(state.createDescription))) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							createMessage: $elm$core$Maybe$Just('Title and description are required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : (($author$project$Sharecrop$Labels$participationUsesReservation(state.createParticipationPolicy) && (($author$project$Sharecrop$Api$reservationHoursValue(state.createReservationHours) < 1) || ($author$project$Sharecrop$Api$reservationHoursValue(state.createReservationHours) > 720))) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							createMessage: $elm$core$Maybe$Just('Reservation expiry must be between 1 and 720 hours.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{createMessage: $elm$core$Maybe$Nothing});
				}),
			$author$project$Sharecrop$Api$postCreateTask(state)));
	});
var $author$project$Sharecrop$Api$credentialsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.credentials;
	} else {
		return _List_Nil;
	}
};
var $author$project$Sharecrop$Types$DeactivateAccountReceived = function (a) {
	return {$: 'DeactivateAccountReceived', a: a};
};
var $author$project$Sharecrop$Api$deactivateAccount = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'DELETE',
		token,
		'/api/account',
		$elm$http$Http$emptyBody,
		$elm$http$Http$expectWhatever($author$project$Sharecrop$Types$DeactivateAccountReceived));
};
var $author$project$Sharecrop$Types$DeactivateMemberReceived = function (a) {
	return {$: 'DeactivateMemberReceived', a: a};
};
var $author$project$Sharecrop$Api$deactivateMemberCommand = F3(
	function (model, state, userId) {
		return (state.activeOrgId === '') ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							provisionMemberMessage: $elm$core$Maybe$Just('Open an organization first.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{provisionMemberMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'PATCH',
				state.accessToken,
				'/api/organizations/' + (state.activeOrgId + ('/members/' + (userId + '/deactivate'))),
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(_List_Nil)),
				$elm$http$Http$expectWhatever($author$project$Sharecrop$Types$DeactivateMemberReceived)));
	});
var $author$project$Sharecrop$Generated$Moderation$ModerationReasonPolicy = {$: 'ModerationReasonPolicy'};
var $author$project$Sharecrop$Labels$participationPolicyTag = function (policy) {
	switch (policy.$) {
		case 'TaskParticipationPolicyOpen':
			return 'open';
		case 'TaskParticipationPolicyReservationRequired':
			return 'reservation_required';
		default:
			return 'approval_required';
	}
};
var $author$project$Main$revisionDraftFor = F2(
	function (taskId, state) {
		return _Utils_eq(
			state.pendingRevisionTaskID,
			$elm$core$Maybe$Just(taskId)) ? state.pendingRevisionResponse : '';
	});
var $author$project$Main$enterPage = F2(
	function (page, state) {
		switch (page.$) {
			case 'TasksPage':
				return _Utils_update(
					state,
					{page: page, taskListOffset: 0, taskListQuery: '', taskListSort: 'newest', taskListTypeFilter: '', taskStateFilter: ''});
			case 'DiscoveryPage':
				return _Utils_update(
					state,
					{discoveryIncludeReserved: false, discoveryOffset: 0, discoveryQuery: '', page: page});
			case 'OrganizationDetailPage':
				var organizationId = page.a;
				return _Utils_update(
					state,
					{
						activeOrgId: organizationId,
						orgAuditEvents: _List_Nil,
						orgBalance: $elm$core$Maybe$Nothing,
						orgCollectibles: _List_Nil,
						orgCollectiblesMessage: $elm$core$Maybe$Nothing,
						orgLedger: _List_Nil,
						orgLedgerOffset: 0,
						orgMembers: _List_Nil,
						orgTaskFilter: '',
						orgTaskMessage: $elm$core$Maybe$Nothing,
						orgTaskOffset: 0,
						orgTaskQuery: '',
						orgTaskSort: 'newest',
						orgTaskTypeFilter: '',
						orgTasks: _List_Nil,
						orgTeamMessage: $elm$core$Maybe$Nothing,
						orgTeams: _List_Nil,
						page: page,
						provisionMemberMessage: $elm$core$Maybe$Nothing,
						provisionMemberRoles: _List_fromArray(
							['member'])
					});
			case 'UserDetailPage':
				return _Utils_update(
					state,
					{page: page, userProfile: $elm$core$Maybe$Nothing, userProfileError: $elm$core$Maybe$Nothing});
			case 'UserWorkPage':
				return _Utils_update(
					state,
					{page: page, userWork: _List_Nil});
			case 'UserSubmissionsPage':
				return _Utils_update(
					state,
					{page: page, userSubmissions: _List_Nil, userSubmissionsOffset: 0});
			case 'SeriesListPage':
				return _Utils_update(
					state,
					{page: page, seriesMessage: $elm$core$Maybe$Nothing});
			case 'SeriesDetailPage':
				return _Utils_update(
					state,
					{addSeriesTaskId: '', page: page, seriesCommentBody: '', seriesDetail: $elm$core$Maybe$Nothing, seriesDetailError: $elm$core$Maybe$Nothing, seriesMessage: $elm$core$Maybe$Nothing, seriesRenameDescription: '', seriesRenameTitle: ''});
			case 'TeamDetailPage':
				return _Utils_update(
					state,
					{page: page, teamCollectibles: _List_Nil, teamCollectiblesMessage: $elm$core$Maybe$Nothing, teamDetail: $elm$core$Maybe$Nothing, teamDetailError: $elm$core$Maybe$Nothing, teamMemberEmail: '', teamMemberMessage: $elm$core$Maybe$Nothing, teamWork: _List_Nil, teamWorkFilter: '', teamWorkMessage: $elm$core$Maybe$Nothing, teamWorkOffset: 0, teamWorkQuery: '', teamWorkSort: 'newest', teamWorkTypeFilter: ''});
			case 'AdminPage':
				return _Utils_update(
					state,
					{adminMessage: $elm$core$Maybe$Nothing, adminModerationOffset: 0, adminModerationReports: _List_Nil, adminModerationResolutionNote: '', adminModerationStateFilter: 'open', adminPrivacyOffset: 0, adminPrivacyRequests: _List_Nil, adminPrivacyResolutionNote: '', adminRetentionRedactedFieldCount: $elm$core$Maybe$Nothing, adminSelectedUserId: '', auditActionFilter: '', auditEvents: _List_Nil, auditEventsOffset: 0, auditSubjectIDFilter: '', auditSubjectKindFilter: '', operations: $elm$core$Maybe$Nothing, page: page, platformAdmins: _List_Nil, platformAdminsOffset: 0});
			case 'InboxPage':
				return _Utils_update(
					state,
					{inboxMessage: $elm$core$Maybe$Nothing, notifications: _List_Nil, notificationsOffset: 0, page: page});
			case 'CollectibleDetailPage':
				return _Utils_update(
					state,
					{page: page, transferMessage: $elm$core$Maybe$Nothing, transferRecipientId: ''});
			case 'TaskDetailPage':
				var taskId = page.a;
				return _Utils_update(
					state,
					{
						activeSubmissionCommentsID: $elm$core$Maybe$Nothing,
						detail: $elm$core$Maybe$Nothing,
						detailError: $elm$core$Maybe$Nothing,
						moderationDetails: '',
						moderationMessage: $elm$core$Maybe$Nothing,
						moderationReason: $author$project$Sharecrop$Generated$Moderation$ModerationReasonPolicy,
						page: page,
						pendingRevisionResponse: '',
						pendingRevisionTaskID: $elm$core$Maybe$Nothing,
						reservationMessage: $elm$core$Maybe$Nothing,
						reservationOrganizationId: '',
						reservationTeamId: '',
						reservations: _List_Nil,
						reviewBan: false,
						reviewMessage: $elm$core$Maybe$Nothing,
						reviewNote: '',
						reviewPartialCredit: '',
						reviewTip: '',
						reviewTipCollectibleId: '',
						submissionCommentBody: '',
						submissionCommentMessage: $elm$core$Maybe$Nothing,
						submissionComments: _List_Nil,
						submissions: _List_Nil,
						submitAttachments: _List_Nil,
						submitInput: A2($author$project$Main$revisionDraftFor, taskId, state),
						submitMessage: $elm$core$Maybe$Nothing,
						taskActionMessage: $elm$core$Maybe$Nothing,
						taskAgentToken: $elm$core$Maybe$Nothing,
						taskCommentBody: '',
						taskCommentMessage: $elm$core$Maybe$Nothing,
						taskComments: _List_Nil,
						taskIntegrationOpen: false
					});
			case 'CollectiblesPage':
				return _Utils_update(
					state,
					{awardDefaultMessage: $elm$core$Maybe$Nothing, awardMessage: $elm$core$Maybe$Nothing, awardRecipientId: '', awardTaskId: '', collectibleMessage: $elm$core$Maybe$Nothing, collectibleName: '', page: page, transferMessage: $elm$core$Maybe$Nothing});
			case 'CreateTaskPage':
				return _Utils_update(
					state,
					{
						createAttachments: _List_Nil,
						createDescription: '',
						createMessage: $elm$core$Maybe$Nothing,
						createParticipationPolicy: $author$project$Sharecrop$Labels$participationPolicyTag($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen),
						createPayloadJson: '',
						createReferenceURL: '',
						createReservationHours: '48',
						createResponseSchema: '{\"kind\":\"freeform\"}',
						createRewardAmount: '',
						createRewardCollectibleIds: _List_Nil,
						createRewardKind: 'none',
						createSchemaFields: _List_Nil,
						createTaskType: 'general',
						createTitle: '',
						page: page
					});
			case 'FundingPage':
				return _Utils_update(
					state,
					{fundMessage: $elm$core$Maybe$Nothing, page: page});
			default:
				return _Utils_update(
					state,
					{page: page});
		}
	});
var $author$project$Sharecrop$Api$entriesFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.entries;
	} else {
		return _List_Nil;
	}
};
var $author$project$Sharecrop$Types$AdminModerationReportsReceived = function (a) {
	return {$: 'AdminModerationReportsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Moderation$ModerationReportsResponse = function (reports) {
	return {reports: reports};
};
var $author$project$Sharecrop$Generated$Moderation$ModerationReportResponse = function (id) {
	return function (subjectKind) {
		return function (subjectID) {
			return function (subjectHref) {
				return function (reason) {
					return function (details) {
						return function (reporterUserID) {
							return function (createdAt) {
								return function (state) {
									return function (resolutionNote) {
										return function (updatedBy) {
											return function (updatedAt) {
												return {createdAt: createdAt, details: details, id: id, reason: reason, reporterUserID: reporterUserID, resolutionNote: resolutionNote, state: state, subjectHref: subjectHref, subjectID: subjectID, subjectKind: subjectKind, updatedAt: updatedAt, updatedBy: updatedBy};
											};
										};
									};
								};
							};
						};
					};
				};
			};
		};
	};
};
var $author$project$Sharecrop$Generated$Moderation$moderationReportResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A5(
			$elm$json$Json$Decode$map4,
			finish,
			A2($elm$json$Json$Decode$field, 'state', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'resolution_note', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'updated_by', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'updated_at', $elm$json$Json$Decode$string));
	},
	A9(
		$elm$json$Json$Decode$map8,
		$author$project$Sharecrop$Generated$Moderation$ModerationReportResponse,
		A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'subject_kind', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'subject_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'subject_href', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'reason', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'details', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'reporter_user_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string)));
var $author$project$Sharecrop$Generated$Moderation$moderationReportsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Moderation$ModerationReportsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'reports',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Moderation$moderationReportResponseDecoder)));
var $elm$url$Url$percentEncode = _Url_percentEncode;
var $author$project$Sharecrop$Api$selectorPageSize = 20;
var $author$project$Sharecrop$Api$fetchAdminModerationReports = F3(
	function (token, stateFilter, offset) {
		var stateQuery = ($elm$core$String$trim(stateFilter) === '') ? '' : ('&state=' + $elm$url$Url$percentEncode(
			$elm$core$String$trim(stateFilter)));
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/admin/moderation/reports?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + ($elm$core$String$fromInt(offset) + stateQuery))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AdminModerationReportsReceived, $author$project$Sharecrop$Generated$Moderation$moderationReportsResponseDecoder));
	});
var $author$project$Sharecrop$Types$AdminPrivacyRequestsReceived = function (a) {
	return {$: 'AdminPrivacyRequestsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Privacy$PrivacyRequestsResponse = function (requests) {
	return {requests: requests};
};
var $author$project$Sharecrop$Generated$Privacy$PrivacyRequestResponse = F9(
	function (id, kind, status, requestedBy, exportJSON, resolutionNote, createdAt, resolvedAt, redactedFieldCount) {
		return {createdAt: createdAt, exportJSON: exportJSON, id: id, kind: kind, redactedFieldCount: redactedFieldCount, requestedBy: requestedBy, resolutionNote: resolutionNote, resolvedAt: resolvedAt, status: status};
	});
var $author$project$Sharecrop$Generated$Privacy$privacyRequestResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A2(
			$elm$json$Json$Decode$map,
			finish,
			A2($elm$json$Json$Decode$field, 'redacted_field_count', $elm$json$Json$Decode$int));
	},
	A9(
		$elm$json$Json$Decode$map8,
		$author$project$Sharecrop$Generated$Privacy$PrivacyRequestResponse,
		A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'kind', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'status', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'requested_by', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'export_json', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'resolution_note', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'resolved_at', $elm$json$Json$Decode$string)));
var $author$project$Sharecrop$Generated$Privacy$privacyRequestsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Privacy$PrivacyRequestsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'requests',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Privacy$privacyRequestResponseDecoder)));
var $author$project$Sharecrop$Api$fetchAdminPrivacyRequests = F2(
	function (token, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/admin/privacy-requests?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AdminPrivacyRequestsReceived, $author$project$Sharecrop$Generated$Privacy$privacyRequestsResponseDecoder));
	});
var $author$project$Sharecrop$Types$AuditEventsReceived = function (a) {
	return {$: 'AuditEventsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Admin$AuditEventsResponse = function (events) {
	return {events: events};
};
var $author$project$Sharecrop$Generated$Admin$AuditEventResponse = F7(
	function (id, actorUserID, action, subjectKind, subjectID, metadataJSON, createdAt) {
		return {action: action, actorUserID: actorUserID, createdAt: createdAt, id: id, metadataJSON: metadataJSON, subjectID: subjectID, subjectKind: subjectKind};
	});
var $elm$json$Json$Decode$map7 = _Json_map7;
var $author$project$Sharecrop$Generated$Admin$auditEventResponseDecoder = A8(
	$elm$json$Json$Decode$map7,
	$author$project$Sharecrop$Generated$Admin$AuditEventResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'actor_user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'action', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'subject_kind', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'subject_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'metadata_json', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Admin$auditEventsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Admin$AuditEventsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'events',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Admin$auditEventResponseDecoder)));
var $author$project$Sharecrop$Api$fetchAuditEvents = F5(
	function (token, actionFilter, subjectKindFilter, subjectIDFilter, offset) {
		var subjectKindQuery = ($elm$core$String$trim(subjectKindFilter) === '') ? '' : ('&subject_kind=' + $elm$url$Url$percentEncode(
			$elm$core$String$trim(subjectKindFilter)));
		var subjectIDQuery = ($elm$core$String$trim(subjectIDFilter) === '') ? '' : ('&subject_id=' + $elm$url$Url$percentEncode(
			$elm$core$String$trim(subjectIDFilter)));
		var actionQuery = ($elm$core$String$trim(actionFilter) === '') ? '' : ('&action=' + $elm$url$Url$percentEncode(
			$elm$core$String$trim(actionFilter)));
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/admin/audit-events?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + ($elm$core$String$fromInt(offset) + (actionQuery + (subjectKindQuery + subjectIDQuery))))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AuditEventsReceived, $author$project$Sharecrop$Generated$Admin$auditEventsResponseDecoder));
	});
var $author$project$Sharecrop$Types$CollectiblesReceived = function (a) {
	return {$: 'CollectiblesReceived', a: a};
};
var $author$project$Sharecrop$Generated$Collectible$CollectiblesResponse = function (collectibles) {
	return {collectibles: collectibles};
};
var $author$project$Sharecrop$Generated$Collectible$collectiblesResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Collectible$CollectiblesResponse,
	A2(
		$elm$json$Json$Decode$field,
		'collectibles',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder)));
var $author$project$Sharecrop$Api$fetchCollectibles = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'GET',
		token,
		'/api/collectibles',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CollectiblesReceived, $author$project$Sharecrop$Generated$Collectible$collectiblesResponseDecoder));
};
var $author$project$Sharecrop$Types$DiscoveryReceived = function (a) {
	return {$: 'DiscoveryReceived', a: a};
};
var $author$project$Sharecrop$Api$boolQuery = function (value) {
	return value ? 'true' : 'false';
};
var $author$project$Sharecrop$Generated$Task$TasksResponse = function (tasks) {
	return {tasks: tasks};
};
var $author$project$Sharecrop$Generated$Task$TaskListItemResponse = function (id) {
	return function (ownerKind) {
		return function (title) {
			return function (rewardKind) {
				return function (rewardCreditAmount) {
					return function (rewardCollectibleCount) {
						return function (participationPolicy) {
							return function (assigneeScope) {
								return function (reservationExpiryHours) {
									return function (state) {
										return function (visibilityKind) {
											return function (availabilityKind) {
												return function (viewerAction) {
													return function (reviewerAction) {
														return function (createdBy) {
															return function (activeAssigneeKind) {
																return function (activeAssigneeID) {
																	return {activeAssigneeID: activeAssigneeID, activeAssigneeKind: activeAssigneeKind, assigneeScope: assigneeScope, availabilityKind: availabilityKind, createdBy: createdBy, id: id, ownerKind: ownerKind, participationPolicy: participationPolicy, reservationExpiryHours: reservationExpiryHours, reviewerAction: reviewerAction, rewardCollectibleCount: rewardCollectibleCount, rewardCreditAmount: rewardCreditAmount, rewardKind: rewardKind, state: state, title: title, viewerAction: viewerAction, visibilityKind: visibilityKind};
																};
															};
														};
													};
												};
											};
										};
									};
								};
							};
						};
					};
				};
			};
		};
	};
};
var $author$project$Sharecrop$Generated$Task$taskListItemResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A2(
			$elm$json$Json$Decode$map,
			finish,
			A2($elm$json$Json$Decode$field, 'active_assignee_id', $elm$json$Json$Decode$string));
	},
	A2(
		$elm$json$Json$Decode$andThen,
		function (finish) {
			return A9(
				$elm$json$Json$Decode$map8,
				finish,
				A2($elm$json$Json$Decode$field, 'reservation_expiry_hours', $elm$json$Json$Decode$int),
				A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Task$taskStateDecoder),
				A2($elm$json$Json$Decode$field, 'visibility_kind', $author$project$Sharecrop$Generated$Task$taskVisibilityKindDecoder),
				A2($elm$json$Json$Decode$field, 'availability_kind', $author$project$Sharecrop$Generated$Task$taskAvailabilityKindDecoder),
				A2($elm$json$Json$Decode$field, 'viewer_action', $author$project$Sharecrop$Generated$Task$taskViewerActionDecoder),
				A2($elm$json$Json$Decode$field, 'reviewer_action', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'active_assignee_kind', $elm$json$Json$Decode$string));
		},
		A9(
			$elm$json$Json$Decode$map8,
			$author$project$Sharecrop$Generated$Task$TaskListItemResponse,
			A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'owner_kind', $author$project$Sharecrop$Generated$Task$taskOwnerKindDecoder),
			A2($elm$json$Json$Decode$field, 'title', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'reward_kind', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'reward_credit_amount', $elm$json$Json$Decode$int),
			A2($elm$json$Json$Decode$field, 'reward_collectible_count', $elm$json$Json$Decode$int),
			A2($elm$json$Json$Decode$field, 'participation_policy', $author$project$Sharecrop$Generated$Task$taskParticipationPolicyDecoder),
			A2($elm$json$Json$Decode$field, 'assignee_scope', $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder))));
var $author$project$Sharecrop$Generated$Task$tasksResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Task$TasksResponse,
	A2(
		$elm$json$Json$Decode$field,
		'tasks',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Task$taskListItemResponseDecoder)));
var $author$project$Sharecrop$Api$fetchDiscovery = F3(
	function (token, includeReserved, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks?scope=public&include_reserved=' + ($author$project$Sharecrop$Api$boolQuery(includeReserved) + ('&limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$DiscoveryReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
	});
var $author$project$Sharecrop$Types$LedgerReceived = function (a) {
	return {$: 'LedgerReceived', a: a};
};
var $author$project$Sharecrop$Generated$Ledger$LedgerResponse = function (entries) {
	return {entries: entries};
};
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryResponse = F4(
	function (id, kind, amount, taskID) {
		return {amount: amount, id: id, kind: kind, taskID: taskID};
	});
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryKindManualAdjustment = {$: 'LedgerEntryKindManualAdjustment'};
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryKindSignupGrant = {$: 'LedgerEntryKindSignupGrant'};
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskEscrow = {$: 'LedgerEntryKindTaskEscrow'};
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskPayout = {$: 'LedgerEntryKindTaskPayout'};
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskRefund = {$: 'LedgerEntryKindTaskRefund'};
var $author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskTip = {$: 'LedgerEntryKindTaskTip'};
var $author$project$Sharecrop$Generated$Ledger$ledgerEntryKindDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'signup_grant':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$LedgerEntryKindSignupGrant);
			case 'task_escrow':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskEscrow);
			case 'task_refund':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskRefund);
			case 'task_payout':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskPayout);
			case 'task_tip':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$LedgerEntryKindTaskTip);
			case 'manual_adjustment':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$LedgerEntryKindManualAdjustment);
			default:
				return $elm$json$Json$Decode$fail('invalid LedgerEntryKind');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Ledger$ledgerEntryResponseDecoder = A5(
	$elm$json$Json$Decode$map4,
	$author$project$Sharecrop$Generated$Ledger$LedgerEntryResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'kind', $author$project$Sharecrop$Generated$Ledger$ledgerEntryKindDecoder),
	A2($elm$json$Json$Decode$field, 'amount', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Ledger$ledgerResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Ledger$LedgerResponse,
	A2(
		$elm$json$Json$Decode$field,
		'entries',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Ledger$ledgerEntryResponseDecoder)));
var $author$project$Sharecrop$Api$fetchLedger = F2(
	function (token, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/credits/ledger?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$LedgerReceived, $author$project$Sharecrop$Generated$Ledger$ledgerResponseDecoder));
	});
var $author$project$Sharecrop$Types$NotificationsReceived = function (a) {
	return {$: 'NotificationsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Notification$NotificationsResponse = function (notifications) {
	return {notifications: notifications};
};
var $author$project$Sharecrop$Generated$Notification$NotificationResponse = F9(
	function (id, recipientUserID, actorUserID, kind, subjectKind, subjectID, state, metadataJSON, createdAt) {
		return {actorUserID: actorUserID, createdAt: createdAt, id: id, kind: kind, metadataJSON: metadataJSON, recipientUserID: recipientUserID, state: state, subjectID: subjectID, subjectKind: subjectKind};
	});
var $author$project$Sharecrop$Generated$Notification$notificationResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A2(
			$elm$json$Json$Decode$map,
			finish,
			A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string));
	},
	A9(
		$elm$json$Json$Decode$map8,
		$author$project$Sharecrop$Generated$Notification$NotificationResponse,
		A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'recipient_user_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'actor_user_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'kind', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'subject_kind', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'subject_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'state', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'metadata_json', $elm$json$Json$Decode$string)));
var $author$project$Sharecrop$Generated$Notification$notificationsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Notification$NotificationsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'notifications',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Notification$notificationResponseDecoder)));
var $author$project$Sharecrop$Api$fetchNotifications = F2(
	function (token, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/notifications?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$NotificationsReceived, $author$project$Sharecrop$Generated$Notification$notificationsResponseDecoder));
	});
var $author$project$Sharecrop$Types$OrgTasksReceived = function (a) {
	return {$: 'OrgTasksReceived', a: a};
};
var $author$project$Sharecrop$Api$taskSearchParams = F4(
	function (queryText, typeFilter, sortOrder, offset) {
		var typePart = (typeFilter === '') ? '' : ('&task_type=' + $elm$url$Url$percentEncode(typeFilter));
		var trimmed = $elm$core$String$trim(queryText);
		var queryPart = (trimmed === '') ? '' : ('&query=' + $elm$url$Url$percentEncode(trimmed));
		var pageQuery = 'limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset)));
		return pageQuery + (queryPart + (typePart + ('&sort=' + $elm$url$Url$percentEncode(sortOrder))));
	});
var $author$project$Sharecrop$Api$fetchOrgTasksPage = F7(
	function (token, organizationId, queryText, stateFilter, typeFilter, sortOrder, offset) {
		var stateQuery = (stateFilter === '') ? '' : ('&state=' + stateFilter);
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks?scope=organization&organization_id=' + (organizationId + ('&' + (A4($author$project$Sharecrop$Api$taskSearchParams, queryText, typeFilter, sortOrder, offset) + stateQuery))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgTasksReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
	});
var $author$project$Sharecrop$Types$OrgTeamsReceived = function (a) {
	return {$: 'OrgTeamsReceived', a: a};
};
var $author$project$Sharecrop$Api$selectorQuery = F3(
	function (queryText, offset, base) {
		var clean = $elm$core$String$trim(queryText);
		var queryPart = (clean === '') ? '' : ('&query=' + $elm$url$Url$percentEncode(clean));
		return base + ('?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + ($elm$core$String$fromInt(offset) + queryPart))));
	});
var $author$project$Sharecrop$Generated$Team$TeamsResponse = function (teams) {
	return {teams: teams};
};
var $author$project$Sharecrop$Generated$Team$teamsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Team$TeamsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'teams',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Team$teamResponseDecoder)));
var $author$project$Sharecrop$Api$fetchOrgTeamsPage = F4(
	function (token, organizationId, queryText, offset) {
		return (organizationId === '') ? $elm$core$Platform$Cmd$none : A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			A3($author$project$Sharecrop$Api$selectorQuery, queryText, offset, '/api/organizations/' + (organizationId + '/teams')),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgTeamsReceived, $author$project$Sharecrop$Generated$Team$teamsResponseDecoder));
	});
var $author$project$Sharecrop$Api$fetchOrgTeams = F2(
	function (token, organizationId) {
		return A4($author$project$Sharecrop$Api$fetchOrgTeamsPage, token, organizationId, '', 0);
	});
var $author$project$Sharecrop$Types$OrgLedgerReceived = function (a) {
	return {$: 'OrgLedgerReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchOrganizationLedgerPage = F3(
	function (token, organizationId, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/organizations/' + (organizationId + ('/credits/ledger?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgLedgerReceived, $author$project$Sharecrop$Generated$Ledger$ledgerResponseDecoder));
	});
var $author$project$Sharecrop$Types$OrganizationsReceived = function (a) {
	return {$: 'OrganizationsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Organization$OrganizationsResponse = function (organizations) {
	return {organizations: organizations};
};
var $author$project$Sharecrop$Generated$Organization$organizationsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Organization$OrganizationsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'organizations',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Organization$organizationResponseDecoder)));
var $author$project$Sharecrop$Api$fetchOrganizationsPage = F3(
	function (token, queryText, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			A3($author$project$Sharecrop$Api$selectorQuery, queryText, offset, '/api/organizations'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrganizationsReceived, $author$project$Sharecrop$Generated$Organization$organizationsResponseDecoder));
	});
var $author$project$Sharecrop$Types$PlatformAdminsReceived = function (a) {
	return {$: 'PlatformAdminsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Admin$PlatformAdminsResponse = function (admins) {
	return {admins: admins};
};
var $author$project$Sharecrop$Generated$Admin$PlatformAdminResponse = F3(
	function (userID, source, createdAt) {
		return {createdAt: createdAt, source: source, userID: userID};
	});
var $author$project$Sharecrop$Generated$Admin$platformAdminResponseDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Generated$Admin$PlatformAdminResponse,
	A2($elm$json$Json$Decode$field, 'user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'source', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Admin$platformAdminsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Admin$PlatformAdminsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'admins',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Admin$platformAdminResponseDecoder)));
var $author$project$Sharecrop$Api$fetchPlatformAdmins = F2(
	function (token, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/admin/platform-admins?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$PlatformAdminsReceived, $author$project$Sharecrop$Generated$Admin$platformAdminsResponseDecoder));
	});
var $author$project$Sharecrop$Types$StandaloneTeamsReceived = function (a) {
	return {$: 'StandaloneTeamsReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchStandaloneTeamsPage = F3(
	function (token, queryText, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			A3($author$project$Sharecrop$Api$selectorQuery, queryText, offset, '/api/teams'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$StandaloneTeamsReceived, $author$project$Sharecrop$Generated$Team$teamsResponseDecoder));
	});
var $author$project$Sharecrop$Types$SubmissionCommentsReceived = function (a) {
	return {$: 'SubmissionCommentsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionCommentsResponse = function (comments) {
	return {comments: comments};
};
var $author$project$Sharecrop$Generated$Submission$submissionCommentsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Submission$SubmissionCommentsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'comments',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionCommentResponseDecoder)));
var $author$project$Sharecrop$Api$fetchSubmissionComments = F2(
	function (token, submissionId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/submissions/' + (submissionId + '/comments'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SubmissionCommentsReceived, $author$project$Sharecrop$Generated$Submission$submissionCommentsResponseDecoder));
	});
var $author$project$Sharecrop$Types$TasksReceived = function (a) {
	return {$: 'TasksReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchTasks = F5(
	function (token, stateFilter, typeFilter, sortOrder, offset) {
		var typeQuery = (typeFilter === '') ? '' : ('&task_type=' + $elm$url$Url$percentEncode(typeFilter));
		var stateQuery = (stateFilter === '') ? '' : ('&state=' + $elm$url$Url$percentEncode(stateFilter));
		var sortQuery = '&sort=' + $elm$url$Url$percentEncode(sortOrder);
		var pageQuery = 'limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset)));
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks?scope=user&' + (pageQuery + (stateQuery + (typeQuery + sortQuery))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TasksReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
	});
var $author$project$Sharecrop$Types$TeamWorkReceived = function (a) {
	return {$: 'TeamWorkReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchTeamWork = F6(
	function (token, teamId, queryText, typeFilter, sortOrder, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/teams/' + (teamId + ('/work?' + A4($author$project$Sharecrop$Api$taskSearchParams, queryText, typeFilter, sortOrder, offset))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TeamWorkReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
	});
var $author$project$Sharecrop$Types$UserDirectoryReceived = function (a) {
	return {$: 'UserDirectoryReceived', a: a};
};
var $author$project$Sharecrop$Types$UserDirectoryEntry = F3(
	function (id, email, status) {
		return {email: email, id: id, status: status};
	});
var $author$project$Sharecrop$Api$userDirectoryEntryDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Types$UserDirectoryEntry,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'email', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'status', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$fetchUserDirectoryPage = F3(
	function (token, queryText, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			A3($author$project$Sharecrop$Api$selectorQuery, queryText, offset, '/api/users'),
			$elm$http$Http$emptyBody,
			A2(
				$elm$http$Http$expectJson,
				$author$project$Sharecrop$Types$UserDirectoryReceived,
				A2(
					$elm$json$Json$Decode$field,
					'users',
					$elm$json$Json$Decode$list($author$project$Sharecrop$Api$userDirectoryEntryDecoder))));
	});
var $author$project$Sharecrop$Types$UserSubmissionsReceived = function (a) {
	return {$: 'UserSubmissionsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionsResponse = function (submissions) {
	return {submissions: submissions};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionResponse = F9(
	function (id, taskID, submitterID, state, responseJSON, reviewNote, attachments, validationErrors, sensitiveFields) {
		return {attachments: attachments, id: id, responseJSON: responseJSON, reviewNote: reviewNote, sensitiveFields: sensitiveFields, state: state, submitterID: submitterID, taskID: taskID, validationErrors: validationErrors};
	});
var $author$project$Sharecrop$Generated$Submission$SubmissionAttachmentResponse = F4(
	function (name, contentType, sizeBytes, dataURL) {
		return {contentType: contentType, dataURL: dataURL, name: name, sizeBytes: sizeBytes};
	});
var $author$project$Sharecrop$Generated$Submission$submissionAttachmentResponseDecoder = A5(
	$elm$json$Json$Decode$map4,
	$author$project$Sharecrop$Generated$Submission$SubmissionAttachmentResponse,
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'content_type', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'size_bytes', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'data_url', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Submission$SubmissionSensitiveFieldResponse = F6(
	function (path, category, retention, redaction, state, redactedAt) {
		return {category: category, path: path, redactedAt: redactedAt, redaction: redaction, retention: retention, state: state};
	});
var $author$project$Sharecrop$Generated$Submission$submissionSensitiveFieldResponseDecoder = A7(
	$elm$json$Json$Decode$map6,
	$author$project$Sharecrop$Generated$Submission$SubmissionSensitiveFieldResponse,
	A2($elm$json$Json$Decode$field, 'path', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'category', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'retention', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'redaction', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'state', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'redacted_at', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Submission$SubmissionStateAccepted = {$: 'SubmissionStateAccepted'};
var $author$project$Sharecrop$Generated$Submission$SubmissionStateChangesRequested = {$: 'SubmissionStateChangesRequested'};
var $author$project$Sharecrop$Generated$Submission$SubmissionStateInvalid = {$: 'SubmissionStateInvalid'};
var $author$project$Sharecrop$Generated$Submission$SubmissionStateRejected = {$: 'SubmissionStateRejected'};
var $author$project$Sharecrop$Generated$Submission$SubmissionStateSubmitted = {$: 'SubmissionStateSubmitted'};
var $author$project$Sharecrop$Generated$Submission$submissionStateDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'submitted':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Submission$SubmissionStateSubmitted);
			case 'invalid':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Submission$SubmissionStateInvalid);
			case 'accepted':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Submission$SubmissionStateAccepted);
			case 'rejected':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Submission$SubmissionStateRejected);
			case 'changes_requested':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Submission$SubmissionStateChangesRequested);
			default:
				return $elm$json$Json$Decode$fail('invalid SubmissionState');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Submission$SubmissionValidationErrorResponse = F2(
	function (path, message) {
		return {message: message, path: path};
	});
var $author$project$Sharecrop$Generated$Submission$submissionValidationErrorResponseDecoder = A3(
	$elm$json$Json$Decode$map2,
	$author$project$Sharecrop$Generated$Submission$SubmissionValidationErrorResponse,
	A2($elm$json$Json$Decode$field, 'path', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'message', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Submission$submissionResponseDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (finish) {
		return A2(
			$elm$json$Json$Decode$map,
			finish,
			A2(
				$elm$json$Json$Decode$field,
				'sensitive_fields',
				$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionSensitiveFieldResponseDecoder)));
	},
	A9(
		$elm$json$Json$Decode$map8,
		$author$project$Sharecrop$Generated$Submission$SubmissionResponse,
		A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'submitter_id', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Submission$submissionStateDecoder),
		A2($elm$json$Json$Decode$field, 'response_json', $elm$json$Json$Decode$string),
		A2($elm$json$Json$Decode$field, 'review_note', $elm$json$Json$Decode$string),
		A2(
			$elm$json$Json$Decode$field,
			'attachments',
			$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionAttachmentResponseDecoder)),
		A2(
			$elm$json$Json$Decode$field,
			'validation_errors',
			$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionValidationErrorResponseDecoder))));
var $author$project$Sharecrop$Generated$Submission$submissionsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Submission$SubmissionsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'submissions',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionResponseDecoder)));
var $author$project$Sharecrop$Api$fetchUserSubmissionsPage = F3(
	function (token, userId, offset) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/users/' + (userId + ('/submissions?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + ('&offset=' + $elm$core$String$fromInt(offset))))),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$UserSubmissionsReceived, $author$project$Sharecrop$Generated$Submission$submissionsResponseDecoder));
	});
var $author$project$Sharecrop$Labels$escrowStateLabel = function (state) {
	switch (state.$) {
		case 'EscrowStateHeld':
			return 'held';
		case 'EscrowStateReleased':
			return 'released';
		default:
			return 'refunded';
	}
};
var $author$project$Sharecrop$View$fundSuccessLabel = function (escrow) {
	return 'Escrowed ' + ($elm$core$String$fromInt(escrow.amount) + (' credits (' + ($author$project$Sharecrop$Labels$escrowStateLabel(escrow.state) + ').')));
};
var $author$project$Sharecrop$Types$FundReceived = function (a) {
	return {$: 'FundReceived', a: a};
};
var $author$project$Sharecrop$Api$fundingRequestBody = F4(
	function (taskId, amount, organizationId, nonce) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'amount',
					$elm$json$Json$Encode$int(amount)),
					_Utils_Tuple2(
					'idempotency_key',
					$elm$json$Json$Encode$string(
						'fund:' + (taskId + (':' + $elm$core$String$fromInt(nonce))))),
					_Utils_Tuple2(
					'organization_id',
					$elm$json$Json$Encode$string(organizationId))
				]));
	});
var $author$project$Sharecrop$Generated$Ledger$TaskEscrowResponse = F3(
	function (taskID, amount, state) {
		return {amount: amount, state: state, taskID: taskID};
	});
var $author$project$Sharecrop$Generated$Ledger$EscrowStateHeld = {$: 'EscrowStateHeld'};
var $author$project$Sharecrop$Generated$Ledger$EscrowStateRefunded = {$: 'EscrowStateRefunded'};
var $author$project$Sharecrop$Generated$Ledger$EscrowStateReleased = {$: 'EscrowStateReleased'};
var $author$project$Sharecrop$Generated$Ledger$escrowStateDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'held':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$EscrowStateHeld);
			case 'released':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$EscrowStateReleased);
			case 'refunded':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Ledger$EscrowStateRefunded);
			default:
				return $elm$json$Json$Decode$fail('invalid EscrowState');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Ledger$taskEscrowResponseDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Generated$Ledger$TaskEscrowResponse,
	A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'amount', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Ledger$escrowStateDecoder));
var $author$project$Sharecrop$Api$postFunding = F5(
	function (token, taskId, amount, organizationId, nonce) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/funding'),
			$elm$http$Http$jsonBody(
				A4($author$project$Sharecrop$Api$fundingRequestBody, taskId, amount, organizationId, nonce)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$FundReceived, $author$project$Sharecrop$Generated$Ledger$taskEscrowResponseDecoder));
	});
var $author$project$Sharecrop$Api$fundTaskCommand = F2(
	function (model, state) {
		var _v0 = $elm$core$String$toInt(state.fundAmount);
		if (_v0.$ === 'Just') {
			var amount = _v0.a;
			return (amount <= 0) ? _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{
								fundMessage: $elm$core$Maybe$Just('Amount must be a positive number of credits.')
							});
					}),
				$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{fundMessage: $elm$core$Maybe$Nothing});
					}),
				A5($author$project$Sharecrop$Api$postFunding, state.accessToken, state.fundTaskId, amount, state.fundOrganizationId, state.fundNonce));
		} else {
			return _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{
								fundMessage: $elm$core$Maybe$Just('Amount must be a whole number of credits.')
							});
					}),
				$elm$core$Platform$Cmd$none);
		}
	});
var $elm$core$Basics$ge = _Utils_ge;
var $author$project$Sharecrop$Types$PlatformAdminGranted = function (a) {
	return {$: 'PlatformAdminGranted', a: a};
};
var $author$project$Sharecrop$Api$grantPlatformAdmin = F2(
	function (token, userID) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/admin/platform-admins',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'user_id',
							$elm$json$Json$Encode$string(userID))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$PlatformAdminGranted, $author$project$Sharecrop$Generated$Admin$platformAdminResponseDecoder));
	});
var $author$project$Sharecrop$Labels$httpErrorLabel = function (error) {
	switch (error.$) {
		case 'BadUrl':
			var url = error.a;
			return 'Bad URL: ' + url;
		case 'Timeout':
			return 'The request timed out.';
		case 'NetworkError':
			return 'A network error occurred.';
		case 'BadStatus':
			var status = error.a;
			return 'The request failed with status ' + ($elm$core$String$fromInt(status) + '.');
		default:
			var message = error.a;
			return 'The response was unexpected: ' + message;
	}
};
var $elm$browser$Browser$Navigation$load = _Browser_load;
var $author$project$Sharecrop$Types$BalanceReceived = function (a) {
	return {$: 'BalanceReceived', a: a};
};
var $author$project$Sharecrop$Generated$Ledger$BalanceResponse = function (amount) {
	return {amount: amount};
};
var $author$project$Sharecrop$Generated$Ledger$balanceResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Ledger$BalanceResponse,
	A2($elm$json$Json$Decode$field, 'amount', $elm$json$Json$Decode$int));
var $author$project$Sharecrop$Api$fetchBalance = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'GET',
		token,
		'/api/credits/balance',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$BalanceReceived, $author$project$Sharecrop$Generated$Ledger$balanceResponseDecoder));
};
var $author$project$Sharecrop$Types$CredentialsReceived = function (a) {
	return {$: 'CredentialsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Agent$AgentCredentialsResponse = function (credentials) {
	return {credentials: credentials};
};
var $author$project$Sharecrop$Generated$Agent$agentCredentialsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Agent$AgentCredentialsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'credentials',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Agent$agentCredentialResponseDecoder)));
var $author$project$Sharecrop$Api$fetchCredentials = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'GET',
		token,
		'/api/agent-credentials',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CredentialsReceived, $author$project$Sharecrop$Generated$Agent$agentCredentialsResponseDecoder));
};
var $author$project$Sharecrop$Api$fetchOrganizations = function (token) {
	return A3($author$project$Sharecrop$Api$fetchOrganizationsPage, token, '', 0);
};
var $author$project$Sharecrop$Types$SavedQueueViewsReceived = function (a) {
	return {$: 'SavedQueueViewsReceived', a: a};
};
var $author$project$Sharecrop$Generated$SavedQueueViews$SavedQueueViewsResponse = function (views) {
	return {views: views};
};
var $author$project$Sharecrop$Generated$SavedQueueViews$SavedQueueViewResponse = F7(
	function (id, scope, name, query, stateFilter, typeFilter, sort) {
		return {id: id, name: name, query: query, scope: scope, sort: sort, stateFilter: stateFilter, typeFilter: typeFilter};
	});
var $author$project$Sharecrop$Generated$SavedQueueViews$savedQueueViewResponseDecoder = A8(
	$elm$json$Json$Decode$map7,
	$author$project$Sharecrop$Generated$SavedQueueViews$SavedQueueViewResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'scope', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'query', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'state_filter', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'type_filter', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'sort', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$SavedQueueViews$savedQueueViewsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$SavedQueueViews$SavedQueueViewsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'views',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$SavedQueueViews$savedQueueViewResponseDecoder)));
var $author$project$Sharecrop$Api$fetchSavedQueueViews = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'GET',
		token,
		'/api/saved-queue-views',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SavedQueueViewsReceived, $author$project$Sharecrop$Generated$SavedQueueViews$savedQueueViewsResponseDecoder));
};
var $author$project$Sharecrop$Api$fetchStandaloneTeams = function (token) {
	return A3($author$project$Sharecrop$Api$fetchStandaloneTeamsPage, token, '', 0);
};
var $author$project$Sharecrop$Api$fetchUserDirectory = function (token) {
	return A3($author$project$Sharecrop$Api$fetchUserDirectoryPage, token, '', 0);
};
var $author$project$Sharecrop$Api$loadAfterAuth = function (token) {
	return $elm$core$Platform$Cmd$batch(
		_List_fromArray(
			[
				$author$project$Sharecrop$Api$fetchBalance(token),
				A2($author$project$Sharecrop$Api$fetchLedger, token, 0),
				A5($author$project$Sharecrop$Api$fetchTasks, token, '', '', 'newest', 0),
				$author$project$Sharecrop$Api$fetchCredentials(token),
				$author$project$Sharecrop$Api$fetchCollectibles(token),
				$author$project$Sharecrop$Api$fetchOrganizations(token),
				$author$project$Sharecrop$Api$fetchUserDirectory(token),
				$author$project$Sharecrop$Api$fetchStandaloneTeams(token),
				$author$project$Sharecrop$Api$fetchSavedQueueViews(token)
			]));
};
var $author$project$Sharecrop$Types$visibilityDefaultTag = 'default';
var $author$project$Main$emptyLoggedIn = function (response) {
	return {
		accessToken: response.accessToken,
		accountEmail: '',
		accountMessage: $elm$core$Maybe$Nothing,
		activeOrgId: '',
		activeSubmissionCommentsID: $elm$core$Maybe$Nothing,
		addSeriesTaskId: '',
		adminMessage: $elm$core$Maybe$Nothing,
		adminModerationOffset: 0,
		adminModerationReports: _List_Nil,
		adminModerationResolutionNote: '',
		adminModerationStateFilter: 'open',
		adminPrivacyOffset: 0,
		adminPrivacyRequests: _List_Nil,
		adminPrivacyResolutionNote: '',
		adminRetentionRedactedFieldCount: $elm$core$Maybe$Nothing,
		adminSelectedUserId: '',
		agentLabel: '',
		agentMessage: $elm$core$Maybe$Nothing,
		agentScopes: _List_fromArray(
			[$author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite]),
		auditActionFilter: '',
		auditEvents: _List_Nil,
		auditEventsOffset: 0,
		auditSubjectIDFilter: '',
		auditSubjectKindFilter: '',
		awardDefaultMessage: $elm$core$Maybe$Nothing,
		awardMessage: $elm$core$Maybe$Nothing,
		awardRecipientId: '',
		awardRecipientKind: 'user',
		awardTaskId: '',
		balance: $elm$core$Maybe$Nothing,
		collectibleCatalog: _List_Nil,
		collectibleKind: $author$project$Sharecrop$Generated$Collectible$CollectibleKindBadge,
		collectibleMessage: $elm$core$Maybe$Nothing,
		collectibleName: '',
		collectiblePolicy: $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyNonTransferableExceptPayout,
		collectibles: _List_Nil,
		createAssigneeScope: $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser,
		createAttachments: _List_Nil,
		createDescription: '',
		createMessage: $elm$core$Maybe$Nothing,
		createOrgName: '',
		createOrgTeamName: '',
		createParticipationPolicy: $author$project$Sharecrop$Labels$participationPolicyTag($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen),
		createPayloadJson: '',
		createReferenceURL: '',
		createReservationHours: '48',
		createResponseSchema: '{\"kind\":\"freeform\"}',
		createRewardAmount: '',
		createRewardCollectibleIds: _List_Nil,
		createRewardKind: 'none',
		createSchemaFields: _List_Nil,
		createScopeOrganizationId: '',
		createScopeTeamId: '',
		createScopeUserId: '',
		createSeriesDescription: '',
		createSeriesTitle: '',
		createTaskOwner: '',
		createTaskType: 'general',
		createTitle: '',
		createVisibility: $author$project$Sharecrop$Types$visibilityDefaultTag,
		credentials: _List_Nil,
		currentPassword: '',
		detail: $elm$core$Maybe$Nothing,
		detailError: $elm$core$Maybe$Nothing,
		discoveryIncludeReserved: false,
		discoveryOffset: 0,
		discoveryQuery: '',
		discoveryTasks: _List_Nil,
		emailVerificationInput: '',
		emailVerificationToken: '',
		entries: _List_Nil,
		fundAmount: '',
		fundMessage: $elm$core$Maybe$Nothing,
		fundNonce: 0,
		fundOrganizationId: '',
		fundTaskId: '',
		inboxMessage: $elm$core$Maybe$Nothing,
		isAdmin: response.role === 'admin',
		ledgerOffset: 0,
		moderationDetails: '',
		moderationMessage: $elm$core$Maybe$Nothing,
		moderationReason: $author$project$Sharecrop$Generated$Moderation$ModerationReasonPolicy,
		newCredential: $elm$core$Maybe$Nothing,
		newPassword: '',
		notifications: _List_Nil,
		notificationsOffset: 0,
		operations: $elm$core$Maybe$Nothing,
		orgAuditEvents: _List_Nil,
		orgBalance: $elm$core$Maybe$Nothing,
		orgCollectibles: _List_Nil,
		orgCollectiblesMessage: $elm$core$Maybe$Nothing,
		orgLedger: _List_Nil,
		orgLedgerOffset: 0,
		orgMembers: _List_Nil,
		orgMessage: $elm$core$Maybe$Nothing,
		orgTaskFilter: '',
		orgTaskMessage: $elm$core$Maybe$Nothing,
		orgTaskOffset: 0,
		orgTaskQuery: '',
		orgTaskSavedViewName: '',
		orgTaskSavedViews: _List_Nil,
		orgTaskSort: 'newest',
		orgTaskTypeFilter: '',
		orgTasks: _List_Nil,
		orgTeamMessage: $elm$core$Maybe$Nothing,
		orgTeamOffset: 0,
		orgTeamQuery: '',
		orgTeams: _List_Nil,
		organizationOffset: 0,
		organizationQuery: '',
		organizations: _List_Nil,
		page: $author$project$Sharecrop$Types$OverviewPage,
		pendingRevisionResponse: '',
		pendingRevisionTaskID: $elm$core$Maybe$Nothing,
		platformAdmins: _List_Nil,
		platformAdminsOffset: 0,
		provisionMemberEmail: '',
		provisionMemberMessage: $elm$core$Maybe$Nothing,
		provisionMemberRoles: _List_fromArray(
			['member']),
		reservationMessage: $elm$core$Maybe$Nothing,
		reservationOrganizationId: '',
		reservationTeamId: '',
		reservations: _List_Nil,
		reviewBan: false,
		reviewMessage: $elm$core$Maybe$Nothing,
		reviewNote: '',
		reviewPartialCredit: '',
		reviewTip: '',
		reviewTipCollectibleId: '',
		seriesCommentBody: '',
		seriesDetail: $elm$core$Maybe$Nothing,
		seriesDetailError: $elm$core$Maybe$Nothing,
		seriesList: _List_Nil,
		seriesMessage: $elm$core$Maybe$Nothing,
		seriesRenameDescription: '',
		seriesRenameTitle: '',
		standaloneTeamOffset: 0,
		standaloneTeamQuery: '',
		standaloneTeams: _List_Nil,
		subjectId: response.subjectID,
		submissionCommentBody: '',
		submissionCommentMessage: $elm$core$Maybe$Nothing,
		submissionComments: _List_Nil,
		submissions: _List_Nil,
		submitAttachments: _List_Nil,
		submitInput: '',
		submitMessage: $elm$core$Maybe$Nothing,
		taskActionMessage: $elm$core$Maybe$Nothing,
		taskAgentToken: $elm$core$Maybe$Nothing,
		taskCommentBody: '',
		taskCommentMessage: $elm$core$Maybe$Nothing,
		taskComments: _List_Nil,
		taskIntegrationOpen: false,
		taskListOffset: 0,
		taskListQuery: '',
		taskListSort: 'newest',
		taskListTypeFilter: '',
		taskStateFilter: '',
		tasks: _List_Nil,
		teamCollectibles: _List_Nil,
		teamCollectiblesMessage: $elm$core$Maybe$Nothing,
		teamDetail: $elm$core$Maybe$Nothing,
		teamDetailError: $elm$core$Maybe$Nothing,
		teamMemberEmail: '',
		teamMemberMessage: $elm$core$Maybe$Nothing,
		teamWork: _List_Nil,
		teamWorkFilter: '',
		teamWorkMessage: $elm$core$Maybe$Nothing,
		teamWorkOffset: 0,
		teamWorkQuery: '',
		teamWorkSavedViewName: '',
		teamWorkSavedViews: _List_Nil,
		teamWorkSort: 'newest',
		teamWorkTypeFilter: '',
		transferMessage: $elm$core$Maybe$Nothing,
		transferRecipientId: '',
		userAgentToken: $elm$core$Maybe$Nothing,
		userDirectory: _List_Nil,
		userDirectoryOffset: 0,
		userDirectoryQuery: '',
		userProfile: $elm$core$Maybe$Nothing,
		userProfileError: $elm$core$Maybe$Nothing,
		userSubmissions: _List_Nil,
		userSubmissionsOffset: 0,
		userWork: _List_Nil
	};
};
var $author$project$Main$loggedInForPage = F2(
	function (response, page) {
		var state = $author$project$Main$emptyLoggedIn(response);
		return A2(
			$author$project$Main$enterPage,
			page,
			_Utils_update(
				state,
				{page: page}));
	});
var $elm$core$Maybe$map = F2(
	function (f, maybe) {
		if (maybe.$ === 'Just') {
			var value = maybe.a;
			return $elm$core$Maybe$Just(
				f(value));
		} else {
			return $elm$core$Maybe$Nothing;
		}
	});
var $author$project$Sharecrop$Types$NotificationReadReceived = function (a) {
	return {$: 'NotificationReadReceived', a: a};
};
var $author$project$Sharecrop$Api$markNotificationRead = F2(
	function (token, notificationId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/notifications/' + (notificationId + '/read'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$NotificationReadReceived, $author$project$Sharecrop$Generated$Notification$notificationResponseDecoder));
	});
var $author$project$Sharecrop$Api$membersFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.members;
	} else {
		return _List_Nil;
	}
};
var $author$project$Sharecrop$Types$MintReceived = function (a) {
	return {$: 'MintReceived', a: a};
};
var $author$project$Sharecrop$Generated$Collectible$collectibleKindEncoder = function (collectibleKind) {
	switch (collectibleKind.$) {
		case 'CollectibleKindUnique':
			return $elm$json$Json$Encode$string('unique');
		case 'CollectibleKindEdition':
			return $elm$json$Json$Encode$string('edition');
		default:
			return $elm$json$Json$Encode$string('badge');
	}
};
var $author$project$Sharecrop$Generated$Collectible$collectibleTransferPolicyEncoder = function (collectibleTransferPolicy) {
	switch (collectibleTransferPolicy.$) {
		case 'CollectibleTransferPolicyNonTransferableExceptPayout':
			return $elm$json$Json$Encode$string('non_transferable_except_payout');
		case 'CollectibleTransferPolicyTransferableBetweenUsers':
			return $elm$json$Json$Encode$string('transferable_between_users');
		case 'CollectibleTransferPolicyTransferableWithinOrganization':
			return $elm$json$Json$Encode$string('transferable_within_organization');
		default:
			return $elm$json$Json$Encode$string('issuer_controlled');
	}
};
var $author$project$Sharecrop$Api$collectibleRequestBody = F3(
	function (name, kind, policy) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'name',
					$elm$json$Json$Encode$string(name)),
					_Utils_Tuple2(
					'kind',
					$author$project$Sharecrop$Generated$Collectible$collectibleKindEncoder(kind)),
					_Utils_Tuple2(
					'transfer_policy',
					$author$project$Sharecrop$Generated$Collectible$collectibleTransferPolicyEncoder(policy))
				]));
	});
var $author$project$Sharecrop$Api$postCollectible = F4(
	function (token, name, kind, policy) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/collectibles',
			$elm$http$Http$jsonBody(
				A3($author$project$Sharecrop$Api$collectibleRequestBody, name, kind, policy)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$MintReceived, $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder));
	});
var $author$project$Sharecrop$Api$mintCommand = F2(
	function (model, state) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.collectibleName)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							collectibleMessage: $elm$core$Maybe$Just('Name is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{collectibleMessage: $elm$core$Maybe$Nothing});
				}),
			A4($author$project$Sharecrop$Api$postCollectible, state.accessToken, state.collectibleName, state.collectibleKind, state.collectiblePolicy));
	});
var $author$project$Sharecrop$View$mintSuccessLabel = function (collectible) {
	return 'Minted ' + (collectible.name + (' (' + ($author$project$Sharecrop$Labels$collectibleStateLabel(collectible.state) + ').')));
};
var $author$project$Sharecrop$Types$TaskTokenMinted = function (a) {
	return {$: 'TaskTokenMinted', a: a};
};
var $author$project$Sharecrop$Api$mintTaskToken = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'POST',
		token,
		'/api/agent-credentials',
		$elm$http$Http$jsonBody(
			A2(
				$author$project$Sharecrop$Api$agentRequestBody,
				'Task worker token',
				_List_fromArray(
					[$author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsRead]))),
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TaskTokenMinted, $author$project$Sharecrop$Generated$Agent$agentCredentialCreatedResponseDecoder));
};
var $author$project$Sharecrop$Types$UserTokenMinted = function (a) {
	return {$: 'UserTokenMinted', a: a};
};
var $author$project$Sharecrop$Api$mintUserToken = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'POST',
		token,
		'/api/agent-credentials',
		$elm$http$Http$jsonBody(
			A2(
				$author$project$Sharecrop$Api$agentRequestBody,
				'Personal agent token',
				_List_fromArray(
					[$author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead, $author$project$Sharecrop$Generated$Agent$AgentScopeTasksWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsRead, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsReview]))),
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$UserTokenMinted, $author$project$Sharecrop$Generated$Agent$agentCredentialCreatedResponseDecoder));
};
var $elm$core$Basics$not = _Basics_not;
var $author$project$Main$orgTaskSavedViewScope = 'organization_tasks';
var $author$project$Main$orgTeamSearchOrganizationID = function (state) {
	return (state.reservationOrganizationId !== '') ? state.reservationOrganizationId : state.activeOrgId;
};
var $author$project$Sharecrop$Generated$Organization$OrganizationMembersResponse = function (members) {
	return {members: members};
};
var $author$project$Sharecrop$Generated$Organization$OrganizationMemberResponse = F5(
	function (id, organizationID, userID, status, roles) {
		return {id: id, organizationID: organizationID, roles: roles, status: status, userID: userID};
	});
var $author$project$Sharecrop$Generated$Organization$MembershipStatusActive = {$: 'MembershipStatusActive'};
var $author$project$Sharecrop$Generated$Organization$MembershipStatusDeactivated = {$: 'MembershipStatusDeactivated'};
var $author$project$Sharecrop$Generated$Organization$MembershipStatusRemoved = {$: 'MembershipStatusRemoved'};
var $author$project$Sharecrop$Generated$Organization$membershipStatusDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'active':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$MembershipStatusActive);
			case 'deactivated':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$MembershipStatusDeactivated);
			case 'removed':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$MembershipStatusRemoved);
			default:
				return $elm$json$Json$Decode$fail('invalid MembershipStatus');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Organization$OrganizationRoleAdmin = {$: 'OrganizationRoleAdmin'};
var $author$project$Sharecrop$Generated$Organization$OrganizationRoleBilling = {$: 'OrganizationRoleBilling'};
var $author$project$Sharecrop$Generated$Organization$OrganizationRoleMember = {$: 'OrganizationRoleMember'};
var $author$project$Sharecrop$Generated$Organization$OrganizationRoleOwner = {$: 'OrganizationRoleOwner'};
var $author$project$Sharecrop$Generated$Organization$OrganizationRolePublicPublisher = {$: 'OrganizationRolePublicPublisher'};
var $author$project$Sharecrop$Generated$Organization$OrganizationRoleReviewer = {$: 'OrganizationRoleReviewer'};
var $author$project$Sharecrop$Generated$Organization$organizationRoleDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'owner':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$OrganizationRoleOwner);
			case 'admin':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$OrganizationRoleAdmin);
			case 'member':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$OrganizationRoleMember);
			case 'billing':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$OrganizationRoleBilling);
			case 'reviewer':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$OrganizationRoleReviewer);
			case 'public_publisher':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Organization$OrganizationRolePublicPublisher);
			default:
				return $elm$json$Json$Decode$fail('invalid OrganizationRole');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Organization$organizationMemberResponseDecoder = A6(
	$elm$json$Json$Decode$map5,
	$author$project$Sharecrop$Generated$Organization$OrganizationMemberResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'organization_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'status', $author$project$Sharecrop$Generated$Organization$membershipStatusDecoder),
	A2(
		$elm$json$Json$Decode$field,
		'roles',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Organization$organizationRoleDecoder)));
var $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Organization$OrganizationMembersResponse,
	A2(
		$elm$json$Json$Decode$field,
		'members',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Organization$organizationMemberResponseDecoder)));
var $author$project$Sharecrop$Api$organizationsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.organizations;
	} else {
		return _List_Nil;
	}
};
var $elm$core$Tuple$pair = F2(
	function (a, b) {
		return _Utils_Tuple2(a, b);
	});
var $author$project$Sharecrop$Types$AddTeamMemberReceived = function (a) {
	return {$: 'AddTeamMemberReceived', a: a};
};
var $author$project$Sharecrop$Generated$Team$TeamDetailResponse = F2(
	function (team, members) {
		return {members: members, team: team};
	});
var $author$project$Sharecrop$Generated$Team$teamDetailResponseDecoder = A3(
	$elm$json$Json$Decode$map2,
	$author$project$Sharecrop$Generated$Team$TeamDetailResponse,
	A2($elm$json$Json$Decode$field, 'team', $author$project$Sharecrop$Generated$Team$teamResponseDecoder),
	A2(
		$elm$json$Json$Decode$field,
		'members',
		$elm$json$Json$Decode$list($elm$json$Json$Decode$string)));
var $author$project$Sharecrop$Api$postAddTeamMember = F3(
	function (token, teamId, email) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/teams/' + (teamId + '/members'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'email',
							$elm$json$Json$Encode$string(email))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AddTeamMemberReceived, $author$project$Sharecrop$Generated$Team$teamDetailResponseDecoder));
	});
var $author$project$Sharecrop$Types$AuthReceived = function (a) {
	return {$: 'AuthReceived', a: a};
};
var $author$project$Sharecrop$Api$authRequestBody = function (model) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'email',
				$elm$json$Json$Encode$string(model.email)),
				_Utils_Tuple2(
				'password',
				$elm$json$Json$Encode$string(model.password))
			]));
};
var $author$project$Sharecrop$Api$postAuth = F2(
	function (url, model) {
		return $elm$http$Http$post(
			{
				body: $elm$http$Http$jsonBody(
					$author$project$Sharecrop$Api$authRequestBody(model)),
				expect: A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AuthReceived, $author$project$Sharecrop$Generated$Auth$authResponseDecoder),
				url: url
			});
	});
var $author$project$Sharecrop$Types$CancelTaskReceived = function (a) {
	return {$: 'CancelTaskReceived', a: a};
};
var $author$project$Sharecrop$Api$postCancelTask = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/cancel'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CancelTaskReceived, $author$project$Sharecrop$Api$taskDetailDecoder));
	});
var $author$project$Sharecrop$Api$postGuest = $elm$http$Http$post(
	{
		body: $elm$http$Http$emptyBody,
		expect: A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AuthReceived, $author$project$Sharecrop$Generated$Auth$authResponseDecoder),
		url: '/api/auth/guest'
	});
var $author$project$Sharecrop$Types$LogoutReceived = function (a) {
	return {$: 'LogoutReceived', a: a};
};
var $author$project$Sharecrop$Api$postLogout = $elm$http$Http$post(
	{
		body: $elm$http$Http$emptyBody,
		expect: $elm$http$Http$expectWhatever($author$project$Sharecrop$Types$LogoutReceived),
		url: '/api/auth/logout'
	});
var $author$project$Sharecrop$Types$OpenTaskReceived = function (a) {
	return {$: 'OpenTaskReceived', a: a};
};
var $author$project$Sharecrop$Api$postOpenTask = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/open'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OpenTaskReceived, $author$project$Sharecrop$Api$taskDetailDecoder));
	});
var $author$project$Sharecrop$Types$RefundCollectibleRewardReceived = function (a) {
	return {$: 'RefundCollectibleRewardReceived', a: a};
};
var $author$project$Sharecrop$Api$postRefundCollectibleReward = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/collectible-refund'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$RefundCollectibleRewardReceived, $author$project$Sharecrop$Generated$Collectible$collectiblesResponseDecoder));
	});
var $author$project$Sharecrop$Types$RefundTaskReceived = function (a) {
	return {$: 'RefundTaskReceived', a: a};
};
var $author$project$Sharecrop$Api$postRefundTask = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/refund'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'idempotency_key',
							$elm$json$Json$Encode$string('ui-refund:' + taskId))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$RefundTaskReceived, $author$project$Sharecrop$Generated$Ledger$taskEscrowResponseDecoder));
	});
var $author$project$Sharecrop$Types$ReservationReceived = function (a) {
	return {$: 'ReservationReceived', a: a};
};
var $author$project$Sharecrop$Api$reservationRequestBody = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Just') {
		var detail = _v0.a;
		var _v1 = detail.assigneeScope;
		switch (_v1.$) {
			case 'TaskAssigneeScopeOrganizationTeam':
				return $elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'assignee_kind',
							$elm$json$Json$Encode$string('organization_team')),
							_Utils_Tuple2(
							'organization_id',
							$elm$json$Json$Encode$string(state.reservationOrganizationId)),
							_Utils_Tuple2(
							'team_id',
							$elm$json$Json$Encode$string(state.reservationTeamId))
						]));
			case 'TaskAssigneeScopeTeam':
				return $elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'assignee_kind',
							$elm$json$Json$Encode$string('team')),
							_Utils_Tuple2(
							'team_id',
							$elm$json$Json$Encode$string(state.reservationTeamId))
						]));
			default:
				return $elm$json$Json$Encode$object(_List_Nil);
		}
	} else {
		return $elm$json$Json$Encode$object(_List_Nil);
	}
};
var $author$project$Sharecrop$Generated$Task$TaskReservationResponse = F6(
	function (id, taskID, assigneeKind, assigneeID, state, requestedBy) {
		return {assigneeID: assigneeID, assigneeKind: assigneeKind, id: id, requestedBy: requestedBy, state: state, taskID: taskID};
	});
var $author$project$Sharecrop$Generated$Task$TaskReservationStateActive = {$: 'TaskReservationStateActive'};
var $author$project$Sharecrop$Generated$Task$TaskReservationStateCancelledByRequester = {$: 'TaskReservationStateCancelledByRequester'};
var $author$project$Sharecrop$Generated$Task$TaskReservationStateCancelledByWorker = {$: 'TaskReservationStateCancelledByWorker'};
var $author$project$Sharecrop$Generated$Task$TaskReservationStateDeclined = {$: 'TaskReservationStateDeclined'};
var $author$project$Sharecrop$Generated$Task$TaskReservationStateExpired = {$: 'TaskReservationStateExpired'};
var $author$project$Sharecrop$Generated$Task$TaskReservationStateRequested = {$: 'TaskReservationStateRequested'};
var $author$project$Sharecrop$Generated$Task$TaskReservationStateSubmitted = {$: 'TaskReservationStateSubmitted'};
var $author$project$Sharecrop$Generated$Task$taskReservationStateDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'requested':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateRequested);
			case 'active':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateActive);
			case 'declined':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateDeclined);
			case 'cancelled_by_requester':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateCancelledByRequester);
			case 'cancelled_by_worker':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateCancelledByWorker);
			case 'expired':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateExpired);
			case 'submitted':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskReservationStateSubmitted);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskReservationState');
		}
	},
	$elm$json$Json$Decode$string);
var $author$project$Sharecrop$Generated$Task$taskReservationResponseDecoder = A7(
	$elm$json$Json$Decode$map6,
	$author$project$Sharecrop$Generated$Task$TaskReservationResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'assignee_kind', $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder),
	A2($elm$json$Json$Decode$field, 'assignee_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Task$taskReservationStateDecoder),
	A2($elm$json$Json$Decode$field, 'requested_by', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$postReservation = F2(
	function (state, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			state.accessToken,
			'/api/tasks/' + (taskId + '/reservations'),
			$elm$http$Http$jsonBody(
				$author$project$Sharecrop$Api$reservationRequestBody(state)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$ReservationReceived, $author$project$Sharecrop$Generated$Task$taskReservationResponseDecoder));
	});
var $author$project$Sharecrop$Types$TaskCommentReceived = function (a) {
	return {$: 'TaskCommentReceived', a: a};
};
var $author$project$Sharecrop$Generated$Task$TaskCommentResponse = F5(
	function (id, taskID, authorUserID, body, createdAt) {
		return {authorUserID: authorUserID, body: body, createdAt: createdAt, id: id, taskID: taskID};
	});
var $author$project$Sharecrop$Generated$Task$taskCommentResponseDecoder = A6(
	$elm$json$Json$Decode$map5,
	$author$project$Sharecrop$Generated$Task$TaskCommentResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'author_user_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'body', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_at', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$postTaskComment = F3(
	function (token, taskId, body) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/comments'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'body',
							$elm$json$Json$Encode$string(body))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TaskCommentReceived, $author$project$Sharecrop$Generated$Task$taskCommentResponseDecoder));
	});
var $author$project$Sharecrop$Types$ProvisionMemberReceived = function (a) {
	return {$: 'ProvisionMemberReceived', a: a};
};
var $author$project$Sharecrop$Api$provisionMemberCommand = F2(
	function (model, state) {
		return ($elm$core$String$isEmpty(
			$elm$core$String$trim(state.provisionMemberEmail)) || (state.activeOrgId === '')) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							provisionMemberMessage: $elm$core$Maybe$Just('A member email is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : ($elm$core$List$isEmpty(state.provisionMemberRoles) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							provisionMemberMessage: $elm$core$Maybe$Just('Select at least one role.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{provisionMemberMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'POST',
				state.accessToken,
				'/api/organizations/' + (state.activeOrgId + '/members'),
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(
						_List_fromArray(
							[
								_Utils_Tuple2(
								'email',
								$elm$json$Json$Encode$string(
									$elm$core$String$trim(state.provisionMemberEmail))),
								_Utils_Tuple2(
								'roles',
								A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, state.provisionMemberRoles))
							]))),
				$elm$http$Http$expectWhatever($author$project$Sharecrop$Types$ProvisionMemberReceived))));
	});
var $elm$browser$Browser$Navigation$pushUrl = _Browser_pushUrl;
var $elm$core$List$head = function (list) {
	if (list.b) {
		var x = list.a;
		var xs = list.b;
		return $elm$core$Maybe$Just(x);
	} else {
		return $elm$core$Maybe$Nothing;
	}
};
var $author$project$Main$queueViewByName = F2(
	function (name, views) {
		return $elm$core$List$head(
			A2(
				$elm$core$List$filter,
				function (view) {
					return _Utils_eq(view.name, name);
				},
				views));
	});
var $author$project$Main$queueViewFromResponse = function (response) {
	return {name: response.name, query: response.query, sort: response.sort, stateFilter: response.stateFilter, typeFilter: response.typeFilter};
};
var $author$project$Sharecrop$Types$CreateAttachmentRejected = function (a) {
	return {$: 'CreateAttachmentRejected', a: a};
};
var $author$project$Sharecrop$Types$CreateAttachmentSelected = F4(
	function (a, b, c, d) {
		return {$: 'CreateAttachmentSelected', a: a, b: b, c: c, d: d};
	});
var $author$project$Main$allowedAttachmentTypes = _List_fromArray(
	['image/png', 'image/jpeg', 'image/gif', 'image/webp', 'text/plain', 'application/json', 'application/pdf']);
var $author$project$Main$attachmentMaxBytes = 500 * 1024;
var $elm$core$List$any = F2(
	function (isOkay, list) {
		any:
		while (true) {
			if (!list.b) {
				return false;
			} else {
				var x = list.a;
				var xs = list.b;
				if (isOkay(x)) {
					return true;
				} else {
					var $temp$isOkay = isOkay,
						$temp$list = xs;
					isOkay = $temp$isOkay;
					list = $temp$list;
					continue any;
				}
			}
		}
	});
var $elm$core$List$member = F2(
	function (x, xs) {
		return A2(
			$elm$core$List$any,
			function (a) {
				return _Utils_eq(a, x);
			},
			xs);
	});
var $elm$time$Time$Posix = function (a) {
	return {$: 'Posix', a: a};
};
var $elm$time$Time$millisToPosix = $elm$time$Time$Posix;
var $elm$file$File$mime = _File_mime;
var $elm$file$File$name = _File_name;
var $elm$file$File$size = _File_size;
var $elm$file$File$toUrl = _File_toUrl;
var $author$project$Main$readAttachment = F3(
	function (file, success, rejected) {
		var sizeBytes = $elm$file$File$size(file);
		var contentType = $elm$file$File$mime(file);
		return (_Utils_cmp(sizeBytes, $author$project$Main$attachmentMaxBytes) > 0) ? $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A2(
					$elm$core$Task$perform,
					$elm$core$Basics$identity,
					$elm$core$Task$succeed(
						rejected('Attachment must be under 500 KiB.')))
				])) : ((!A2($elm$core$List$member, contentType, $author$project$Main$allowedAttachmentTypes)) ? $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A2(
					$elm$core$Task$perform,
					$elm$core$Basics$identity,
					$elm$core$Task$succeed(
						rejected('Attachment type is not allowed.')))
				])) : A2(
			$elm$core$Task$perform,
			A3(
				success,
				$elm$file$File$name(file),
				contentType,
				sizeBytes),
			$elm$file$File$toUrl(file)));
	});
var $author$project$Main$readCreateAttachment = function (file) {
	return A3($author$project$Main$readAttachment, file, $author$project$Sharecrop$Types$CreateAttachmentSelected, $author$project$Sharecrop$Types$CreateAttachmentRejected);
};
var $author$project$Sharecrop$Types$SubmitAttachmentRejected = function (a) {
	return {$: 'SubmitAttachmentRejected', a: a};
};
var $author$project$Sharecrop$Types$SubmitAttachmentSelected = F4(
	function (a, b, c, d) {
		return {$: 'SubmitAttachmentSelected', a: a, b: b, c: c, d: d};
	});
var $author$project$Main$readSubmitAttachment = function (file) {
	return A3($author$project$Main$readAttachment, file, $author$project$Sharecrop$Types$SubmitAttachmentSelected, $author$project$Sharecrop$Types$SubmitAttachmentRejected);
};
var $author$project$Sharecrop$Types$DetailReceived = function (a) {
	return {$: 'DetailReceived', a: a};
};
var $author$project$Sharecrop$Api$publicTaskDetailFromResponse = function (response) {
	return $author$project$Sharecrop$Api$taskDetailFromResponse(response);
};
var $author$project$Sharecrop$Api$publicTaskDetailDecoder = A2($elm$json$Json$Decode$map, $author$project$Sharecrop$Api$publicTaskDetailFromResponse, $author$project$Sharecrop$Generated$Task$taskResponseDecoder);
var $author$project$Sharecrop$Api$fetchPublicTaskDetail = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + taskId,
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$DetailReceived, $author$project$Sharecrop$Api$publicTaskDetailDecoder));
	});
var $author$project$Sharecrop$Types$ReservationsReceived = function (a) {
	return {$: 'ReservationsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Task$TaskReservationsResponse = function (reservations) {
	return {reservations: reservations};
};
var $author$project$Sharecrop$Generated$Task$taskReservationsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Task$TaskReservationsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'reservations',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Task$taskReservationResponseDecoder)));
var $author$project$Sharecrop$Api$fetchReservations = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + (taskId + '/reservations'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$ReservationsReceived, $author$project$Sharecrop$Generated$Task$taskReservationsResponseDecoder));
	});
var $author$project$Sharecrop$Types$SubmissionsReceived = function (a) {
	return {$: 'SubmissionsReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchSubmissions = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + (taskId + '/submissions'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SubmissionsReceived, $author$project$Sharecrop$Generated$Submission$submissionsResponseDecoder));
	});
var $author$project$Sharecrop$Api$refreshAfterAccept = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		var _v1 = state.page;
		if (_v1.$ === 'TaskDetailPage') {
			var taskId = _v1.a;
			return $elm$core$Platform$Cmd$batch(
				_List_fromArray(
					[
						A2($author$project$Sharecrop$Api$fetchSubmissions, state.accessToken, taskId),
						$author$project$Sharecrop$Api$fetchBalance(state.accessToken),
						A2($author$project$Sharecrop$Api$fetchPublicTaskDetail, state.accessToken, taskId),
						A2($author$project$Sharecrop$Api$fetchReservations, state.accessToken, taskId)
					]));
		} else {
			return $elm$core$Platform$Cmd$none;
		}
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshCollectibles = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $author$project$Sharecrop$Api$fetchCollectibles(state.accessToken);
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshCredentials = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $author$project$Sharecrop$Api$fetchCredentials(state.accessToken);
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshDetailReservations = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		var _v1 = state.page;
		if (_v1.$ === 'TaskDetailPage') {
			var taskId = _v1.a;
			return $elm$core$Platform$Cmd$batch(
				_List_fromArray(
					[
						A2($author$project$Sharecrop$Api$fetchPublicTaskDetail, state.accessToken, taskId),
						A2($author$project$Sharecrop$Api$fetchReservations, state.accessToken, taskId)
					]));
		} else {
			return $elm$core$Platform$Cmd$none;
		}
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$fetchUserSubmissions = F2(
	function (token, userId) {
		return A3($author$project$Sharecrop$Api$fetchUserSubmissionsPage, token, userId, 0);
	});
var $author$project$Sharecrop$Api$refreshDetailSubmissions = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		var _v1 = state.page;
		if (_v1.$ === 'TaskDetailPage') {
			var taskId = _v1.a;
			return $elm$core$Platform$Cmd$batch(
				_List_fromArray(
					[
						A2($author$project$Sharecrop$Api$fetchSubmissions, state.accessToken, taskId),
						A2($author$project$Sharecrop$Api$fetchUserSubmissions, state.accessToken, state.subjectId)
					]));
		} else {
			return $elm$core$Platform$Cmd$none;
		}
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshLedger = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					$author$project$Sharecrop$Api$fetchBalance(state.accessToken),
					A2($author$project$Sharecrop$Api$fetchLedger, state.accessToken, state.ledgerOffset)
				]));
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshOrganizations = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $author$project$Sharecrop$Api$fetchOrganizations(state.accessToken);
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshTasksAndDiscovery = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, state.taskListTypeFilter, state.taskListSort, state.taskListOffset),
					A3($author$project$Sharecrop$Api$fetchDiscovery, state.accessToken, state.discoveryIncludeReserved, state.discoveryOffset)
				]));
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$refreshTasksAndLedger = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, state.taskListTypeFilter, state.taskListSort, state.taskListOffset),
					$author$project$Sharecrop$Api$fetchBalance(state.accessToken),
					A2($author$project$Sharecrop$Api$fetchLedger, state.accessToken, state.ledgerOffset)
				]));
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $elm$json$Json$Encode$bool = _Json_wrap;
var $author$project$Sharecrop$Api$rejectRequestBody = F5(
	function (submissionId, reviewNote, partialCredit, tipAmount, banImplementor) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'idempotency_key',
					$elm$json$Json$Encode$string('ui-reject:' + submissionId)),
					_Utils_Tuple2(
					'review_note',
					$elm$json$Json$Encode$string(reviewNote)),
					_Utils_Tuple2(
					'partial_credit_amount',
					$elm$json$Json$Encode$int(
						$author$project$Sharecrop$Api$intInputOrZero(partialCredit))),
					_Utils_Tuple2(
					'tip_amount',
					$elm$json$Json$Encode$int(
						$author$project$Sharecrop$Api$intInputOrZero(tipAmount))),
					_Utils_Tuple2(
					'ban_implementor',
					$elm$json$Json$Encode$bool(banImplementor))
				]));
	});
var $author$project$Sharecrop$Api$postReject = F7(
	function (token, taskId, submissionId, reviewNote, partialCredit, tipAmount, banImplementor) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/submissions/' + (submissionId + '/reject'))),
			$elm$http$Http$jsonBody(
				A5($author$project$Sharecrop$Api$rejectRequestBody, submissionId, reviewNote, partialCredit, tipAmount, banImplementor)),
			$elm$http$Http$expectWhatever(
				$author$project$Sharecrop$Types$ReviewActionReceived(submissionId)));
	});
var $author$project$Sharecrop$Api$rejectCommand = F3(
	function (model, state, submissionId) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reviewMessage: $elm$core$Maybe$Nothing});
					}),
				A7($author$project$Sharecrop$Api$postReject, state.accessToken, taskId, submissionId, state.reviewNote, state.reviewPartialCredit, state.reviewTip, state.reviewBan));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $elm$json$Json$Encode$null = _Json_encodeNull;
var $author$project$Main$reloadDemo = _Platform_outgoingPort(
	'reloadDemo',
	function ($) {
		return $elm$json$Json$Encode$null;
	});
var $elm$core$Tuple$second = function (_v0) {
	var y = _v0.b;
	return y;
};
var $author$project$Main$removeAt = F2(
	function (index, values) {
		return A2(
			$elm$core$List$map,
			$elm$core$Tuple$second,
			A2(
				$elm$core$List$filter,
				function (_v0) {
					var currentIndex = _v0.a;
					return !_Utils_eq(currentIndex, index);
				},
				A2($elm$core$List$indexedMap, $elm$core$Tuple$pair, values)));
	});
var $author$project$Main$removePlatformAdmin = F2(
	function (userID, admins) {
		return A2(
			$elm$core$List$filter,
			function (admin) {
				return !_Utils_eq(admin.userID, userID);
			},
			admins);
	});
var $author$project$Sharecrop$Api$removeSeriesTaskCommand = F3(
	function (token, seriesId, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'DELETE',
			token,
			'/api/task-series/' + (seriesId + ('/tasks/' + taskId)),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesMutationReceived, $author$project$Sharecrop$Api$seriesDetailDecoder));
	});
var $author$project$Main$replaceModerationReport = F2(
	function (replacement, reports) {
		return A2(
			$elm$core$List$map,
			function (report) {
				return _Utils_eq(report.id, replacement.id) ? replacement : report;
			},
			reports);
	});
var $author$project$Main$replaceNotification = F2(
	function (replacement, notifications) {
		return A2(
			$elm$core$List$map,
			function (notification) {
				return _Utils_eq(notification.id, replacement.id) ? replacement : notification;
			},
			notifications);
	});
var $author$project$Main$replacePrivacyRequest = F2(
	function (replacement, requests) {
		return A2(
			$elm$core$List$map,
			function (request) {
				return _Utils_eq(request.id, replacement.id) ? replacement : request;
			},
			requests);
	});
var $author$project$Sharecrop$Types$ModerationReportReceived = function (a) {
	return {$: 'ModerationReportReceived', a: a};
};
var $author$project$Sharecrop$Generated$Moderation$moderationReasonEncoder = function (moderationReason) {
	switch (moderationReason.$) {
		case 'ModerationReasonSpam':
			return $elm$json$Json$Encode$string('spam');
		case 'ModerationReasonAbuse':
			return $elm$json$Json$Encode$string('abuse');
		case 'ModerationReasonPII':
			return $elm$json$Json$Encode$string('pii');
		case 'ModerationReasonPolicy':
			return $elm$json$Json$Encode$string('policy');
		default:
			return $elm$json$Json$Encode$string('other');
	}
};
var $author$project$Sharecrop$Api$reportTask = F4(
	function (token, taskId, reason, details) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/moderation/reports',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'subject_kind',
							$elm$json$Json$Encode$string('task')),
							_Utils_Tuple2(
							'subject_id',
							$elm$json$Json$Encode$string(taskId)),
							_Utils_Tuple2(
							'reason',
							$author$project$Sharecrop$Generated$Moderation$moderationReasonEncoder(reason)),
							_Utils_Tuple2(
							'details',
							$elm$json$Json$Encode$string(details))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$ModerationReportReceived, $author$project$Sharecrop$Generated$Moderation$moderationReportResponseDecoder));
	});
var $author$project$Sharecrop$Api$requestChangesBody = function (reviewNote) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'review_note',
				$elm$json$Json$Encode$string(reviewNote))
			]));
};
var $author$project$Sharecrop$Api$postRequestChanges = F4(
	function (token, taskId, submissionId, reviewNote) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/submissions/' + (submissionId + '/request-changes'))),
			$elm$http$Http$jsonBody(
				$author$project$Sharecrop$Api$requestChangesBody(reviewNote)),
			$elm$http$Http$expectWhatever(
				$author$project$Sharecrop$Types$ReviewActionReceived(submissionId)));
	});
var $author$project$Sharecrop$Api$requestChangesCommand = F3(
	function (model, state, submissionId) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reviewMessage: $elm$core$Maybe$Nothing});
					}),
				A4($author$project$Sharecrop$Api$postRequestChanges, state.accessToken, taskId, submissionId, state.reviewNote));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Sharecrop$Types$EmailVerificationRequested = function (a) {
	return {$: 'EmailVerificationRequested', a: a};
};
var $elm$json$Json$Decode$oneOf = _Json_oneOf;
var $author$project$Sharecrop$Api$tokenDecoder = $elm$json$Json$Decode$oneOf(
	_List_fromArray(
		[
			A2($elm$json$Json$Decode$field, 'token', $elm$json$Json$Decode$string),
			A2(
			$elm$json$Json$Decode$map,
			function (_v0) {
				return '';
			},
			A2($elm$json$Json$Decode$field, 'status', $elm$json$Json$Decode$string))
		]));
var $author$project$Sharecrop$Api$requestEmailVerification = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'POST',
		token,
		'/api/account/email-verification',
		$elm$http$Http$jsonBody(
			$elm$json$Json$Encode$object(_List_Nil)),
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$EmailVerificationRequested, $author$project$Sharecrop$Api$tokenDecoder));
};
var $author$project$Sharecrop$Types$PasswordResetRequested = function (a) {
	return {$: 'PasswordResetRequested', a: a};
};
var $author$project$Sharecrop$Api$requestPasswordReset = function (model) {
	return $elm$http$Http$post(
		{
			body: $elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'email',
							$elm$json$Json$Encode$string(model.resetEmail))
						]))),
			expect: A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$PasswordResetRequested, $author$project$Sharecrop$Api$tokenDecoder),
			url: '/api/auth/password-reset/request'
		});
};
var $author$project$Sharecrop$Types$PrivacyRequestReceived = function (a) {
	return {$: 'PrivacyRequestReceived', a: a};
};
var $author$project$Sharecrop$Generated$Privacy$privacyRequestKindEncoder = function (privacyRequestKind) {
	if (privacyRequestKind.$ === 'PrivacyRequestKindDataExport') {
		return $elm$json$Json$Encode$string('data_export');
	} else {
		return $elm$json$Json$Encode$string('sensitive_field_deletion');
	}
};
var $author$project$Sharecrop$Api$requestPrivacy = F2(
	function (token, kind) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/privacy-requests',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'kind',
							$author$project$Sharecrop$Generated$Privacy$privacyRequestKindEncoder(kind))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$PrivacyRequestReceived, $author$project$Sharecrop$Generated$Privacy$privacyRequestResponseDecoder));
	});
var $author$project$Sharecrop$Types$ReservationChangeReceived = function (a) {
	return {$: 'ReservationChangeReceived', a: a};
};
var $author$project$Sharecrop$Api$postReservationChange = F4(
	function (token, taskId, reservationId, action) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/reservations/' + (reservationId + ('/' + action)))),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$ReservationChangeReceived, $author$project$Sharecrop$Generated$Task$taskReservationResponseDecoder));
	});
var $author$project$Sharecrop$Api$reservationChangeCommand = F4(
	function (model, state, reservationId, action) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reservationMessage: $elm$core$Maybe$Nothing});
					}),
				A4($author$project$Sharecrop$Api$postReservationChange, state.accessToken, taskId, reservationId, action));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Sharecrop$Labels$reservationStateLabel = function (state) {
	switch (state.$) {
		case 'TaskReservationStateRequested':
			return 'requested';
		case 'TaskReservationStateActive':
			return 'active';
		case 'TaskReservationStateDeclined':
			return 'declined';
		case 'TaskReservationStateCancelledByRequester':
			return 'cancelled by requester';
		case 'TaskReservationStateCancelledByWorker':
			return 'cancelled by worker';
		case 'TaskReservationStateExpired':
			return 'expired';
		default:
			return 'submitted';
	}
};
var $author$project$Sharecrop$View$reservationSuccessLabel = function (reservation) {
	return 'Reservation ' + ($author$project$Sharecrop$Labels$reservationStateLabel(reservation.state) + '.');
};
var $author$project$Sharecrop$Types$AdminPrivacyRequestResolved = function (a) {
	return {$: 'AdminPrivacyRequestResolved', a: a};
};
var $author$project$Sharecrop$Api$resolveAdminPrivacyRequest = F3(
	function (token, requestId, resolutionNote) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/admin/privacy-requests/' + (requestId + '/resolve'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'resolution_note',
							$elm$json$Json$Encode$string(resolutionNote))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AdminPrivacyRequestResolved, $author$project$Sharecrop$Generated$Privacy$privacyRequestResponseDecoder));
	});
var $author$project$Sharecrop$Types$AgentRevoked = function (a) {
	return {$: 'AgentRevoked', a: a};
};
var $author$project$Sharecrop$Api$revokeAgent = F2(
	function (token, credentialId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/agent-credentials/' + (credentialId + '/revoke'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AgentRevoked, $author$project$Sharecrop$Generated$Agent$agentCredentialResponseDecoder));
	});
var $author$project$Sharecrop$Types$PlatformAdminRevoked = function (a) {
	return {$: 'PlatformAdminRevoked', a: a};
};
var $author$project$Sharecrop$Api$revokePlatformAdmin = F2(
	function (token, userID) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/admin/platform-admins/' + (userID + '/revoke'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$PlatformAdminRevoked, $author$project$Sharecrop$Generated$Admin$platformAdminResponseDecoder));
	});
var $author$project$Sharecrop$Types$OperationsReceived = function (a) {
	return {$: 'OperationsReceived', a: a};
};
var $author$project$Sharecrop$Types$TeamDetailReceived = function (a) {
	return {$: 'TeamDetailReceived', a: a};
};
var $author$project$Sharecrop$Types$UserWorkReceived = function (a) {
	return {$: 'UserWorkReceived', a: a};
};
var $author$project$Sharecrop$Types$CollectibleCatalogReceived = function (a) {
	return {$: 'CollectibleCatalogReceived', a: a};
};
var $author$project$Sharecrop$Generated$Collectible$CollectibleCatalogResponse = function (entries) {
	return {entries: entries};
};
var $author$project$Sharecrop$Generated$Collectible$CollectibleCatalogEntry = F5(
	function (slug, name, kind, transferPolicy, art) {
		return {art: art, kind: kind, name: name, slug: slug, transferPolicy: transferPolicy};
	});
var $author$project$Sharecrop$Generated$Collectible$collectibleCatalogEntryDecoder = A6(
	$elm$json$Json$Decode$map5,
	$author$project$Sharecrop$Generated$Collectible$CollectibleCatalogEntry,
	A2($elm$json$Json$Decode$field, 'slug', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'kind', $author$project$Sharecrop$Generated$Collectible$collectibleKindDecoder),
	A2($elm$json$Json$Decode$field, 'transfer_policy', $author$project$Sharecrop$Generated$Collectible$collectibleTransferPolicyDecoder),
	A2($elm$json$Json$Decode$field, 'art', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Generated$Collectible$collectibleCatalogResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Collectible$CollectibleCatalogResponse,
	A2(
		$elm$json$Json$Decode$field,
		'entries',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Collectible$collectibleCatalogEntryDecoder)));
var $author$project$Sharecrop$Api$fetchCollectibleCatalog = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'GET',
		token,
		'/api/collectibles/catalog',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$CollectibleCatalogReceived, $author$project$Sharecrop$Generated$Collectible$collectibleCatalogResponseDecoder));
};
var $author$project$Sharecrop$Types$TaskCommentsReceived = function (a) {
	return {$: 'TaskCommentsReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchTaskComments = F2(
	function (token, taskId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + (taskId + '/comments'),
			$elm$http$Http$emptyBody,
			A2(
				$elm$http$Http$expectJson,
				$author$project$Sharecrop$Types$TaskCommentsReceived,
				A2(
					$elm$json$Json$Decode$field,
					'comments',
					$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Task$taskCommentResponseDecoder))));
	});
var $author$project$Sharecrop$Api$fetchDetailCommands = F3(
	function (token, subjectId, taskId) {
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A2($author$project$Sharecrop$Api$fetchPublicTaskDetail, token, taskId),
					A2($author$project$Sharecrop$Api$fetchSubmissions, token, taskId),
					A2($author$project$Sharecrop$Api$fetchReservations, token, taskId),
					A2($author$project$Sharecrop$Api$fetchTaskComments, token, taskId),
					A2($author$project$Sharecrop$Api$fetchUserSubmissions, token, subjectId),
					$author$project$Sharecrop$Api$fetchOrganizations(token)
				]));
	});
var $author$project$Sharecrop$Types$OrgCollectiblesReceived = function (a) {
	return {$: 'OrgCollectiblesReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchOrganizationCollectibles = F2(
	function (token, orgId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/organizations/' + (orgId + '/collectibles'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgCollectiblesReceived, $author$project$Sharecrop$Generated$Collectible$collectiblesResponseDecoder));
	});
var $author$project$Sharecrop$Types$SeriesDetailReceived = function (a) {
	return {$: 'SeriesDetailReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchSeriesDetail = F2(
	function (token, seriesId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/task-series/' + seriesId,
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesDetailReceived, $author$project$Sharecrop$Api$seriesDetailDecoder));
	});
var $author$project$Sharecrop$Types$SeriesListReceived = function (a) {
	return {$: 'SeriesListReceived', a: a};
};
var $author$project$Sharecrop$Generated$TaskSeries$TaskSeriesListResponse = function (series) {
	return {series: series};
};
var $author$project$Sharecrop$Generated$TaskSeries$taskSeriesListResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$TaskSeries$TaskSeriesListResponse,
	A2(
		$elm$json$Json$Decode$field,
		'series',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$TaskSeries$taskSeriesResponseDecoder)));
var $author$project$Sharecrop$Api$fetchSeriesList = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'GET',
		token,
		'/api/task-series',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesListReceived, $author$project$Sharecrop$Generated$TaskSeries$taskSeriesListResponseDecoder));
};
var $author$project$Sharecrop$Types$TeamCollectiblesReceived = function (a) {
	return {$: 'TeamCollectiblesReceived', a: a};
};
var $author$project$Sharecrop$Api$fetchTeamCollectibles = F2(
	function (token, teamId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/teams/' + (teamId + '/collectibles'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TeamCollectiblesReceived, $author$project$Sharecrop$Generated$Collectible$collectiblesResponseDecoder));
	});
var $author$project$Sharecrop$Types$UserProfileReceived = function (a) {
	return {$: 'UserProfileReceived', a: a};
};
var $author$project$Sharecrop$Generated$Task$UserProfileResponse = F2(
	function (id, tasks) {
		return {id: id, tasks: tasks};
	});
var $author$project$Sharecrop$Generated$Task$userProfileResponseDecoder = A3(
	$elm$json$Json$Decode$map2,
	$author$project$Sharecrop$Generated$Task$UserProfileResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2(
		$elm$json$Json$Decode$field,
		'tasks',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Task$taskListItemResponseDecoder)));
var $author$project$Sharecrop$Api$fetchUserProfile = F2(
	function (token, userId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'GET',
			token,
			'/api/users/' + userId,
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$UserProfileReceived, $author$project$Sharecrop$Generated$Task$userProfileResponseDecoder));
	});
var $author$project$Sharecrop$Types$OrgAuditEventsReceived = function (a) {
	return {$: 'OrgAuditEventsReceived', a: a};
};
var $author$project$Sharecrop$Types$OrgBalanceReceived = function (a) {
	return {$: 'OrgBalanceReceived', a: a};
};
var $author$project$Sharecrop$Api$loadOrganization = F2(
	function (token, organizationId) {
		return (organizationId === '') ? $elm$core$Platform$Cmd$none : $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A5(
					$author$project$Sharecrop$Api$authorizedRequest,
					'GET',
					token,
					'/api/organizations/' + (organizationId + '/credits/balance'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgBalanceReceived, $author$project$Sharecrop$Generated$Ledger$balanceResponseDecoder)),
					A3($author$project$Sharecrop$Api$fetchOrganizationLedgerPage, token, organizationId, 0),
					A5(
					$author$project$Sharecrop$Api$authorizedRequest,
					'GET',
					token,
					'/api/organizations/' + (organizationId + ('/audit-events?limit=' + ($elm$core$String$fromInt($author$project$Sharecrop$Api$selectorPageSize) + '&offset=0'))),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgAuditEventsReceived, $author$project$Sharecrop$Generated$Admin$auditEventsResponseDecoder)),
					A2($author$project$Sharecrop$Api$fetchOrgTeams, token, organizationId),
					A5(
					$author$project$Sharecrop$Api$authorizedRequest,
					'GET',
					token,
					'/api/organizations/' + (organizationId + '/members'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgMembersReceived, $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder)),
					A7($author$project$Sharecrop$Api$fetchOrgTasksPage, token, organizationId, '', '', '', 'newest', 0)
				]));
	});
var $author$project$Sharecrop$Generated$Admin$OperationsResponse = F8(
	function (status, accountTokenDelivery, mcpStorage, rateLimitStorage, activeMCPSessions, activeIPRateBuckets, activeSubjectRateBuckets, secureCookies) {
		return {accountTokenDelivery: accountTokenDelivery, activeIPRateBuckets: activeIPRateBuckets, activeMCPSessions: activeMCPSessions, activeSubjectRateBuckets: activeSubjectRateBuckets, mcpStorage: mcpStorage, rateLimitStorage: rateLimitStorage, secureCookies: secureCookies, status: status};
	});
var $author$project$Sharecrop$Generated$Admin$operationsResponseDecoder = A9(
	$elm$json$Json$Decode$map8,
	$author$project$Sharecrop$Generated$Admin$OperationsResponse,
	A2($elm$json$Json$Decode$field, 'status', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'account_token_delivery', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'mcp_storage', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'rate_limit_storage', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'active_mcp_sessions', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'active_ip_rate_buckets', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'active_subject_rate_buckets', $elm$json$Json$Decode$int),
	A2($elm$json$Json$Decode$field, 'secure_cookies', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$routeLoadCmd = F3(
	function (token, subjectId, page) {
		switch (page.$) {
			case 'OverviewPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Sharecrop$Api$fetchBalance(token),
							A2($author$project$Sharecrop$Api$fetchLedger, token, 0)
						]));
			case 'TasksPage':
				return A5($author$project$Sharecrop$Api$fetchTasks, token, '', '', 'newest', 0);
			case 'CreateTaskPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Sharecrop$Api$fetchOrganizations(token),
							$author$project$Sharecrop$Api$fetchCollectibles(token),
							$author$project$Sharecrop$Api$fetchUserDirectory(token),
							$author$project$Sharecrop$Api$fetchStandaloneTeams(token)
						]));
			case 'TaskDetailPage':
				var taskId = page.a;
				return A3($author$project$Sharecrop$Api$fetchDetailCommands, token, subjectId, taskId);
			case 'DiscoveryPage':
				return A3($author$project$Sharecrop$Api$fetchDiscovery, token, false, 0);
			case 'FundingPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							A5($author$project$Sharecrop$Api$fetchTasks, token, '', '', 'newest', 0),
							$author$project$Sharecrop$Api$fetchOrganizations(token)
						]));
			case 'AgentsPage':
				return $author$project$Sharecrop$Api$fetchCredentials(token);
			case 'CollectiblesPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Sharecrop$Api$fetchCollectibles(token),
							$author$project$Sharecrop$Api$fetchCollectibleCatalog(token),
							A5($author$project$Sharecrop$Api$fetchTasks, token, '', '', 'newest', 0),
							$author$project$Sharecrop$Api$fetchOrganizations(token)
						]));
			case 'OrganizationsPage':
				return $author$project$Sharecrop$Api$fetchOrganizations(token);
			case 'OrganizationDetailPage':
				var organizationId = page.a;
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Sharecrop$Api$fetchOrganizations(token),
							A2($author$project$Sharecrop$Api$loadOrganization, token, organizationId),
							A2($author$project$Sharecrop$Api$fetchOrganizationCollectibles, token, organizationId)
						]));
			case 'UserDetailPage':
				var userId = page.a;
				return A2($author$project$Sharecrop$Api$fetchUserProfile, token, userId);
			case 'UserWorkPage':
				var userId = page.a;
				return A5(
					$author$project$Sharecrop$Api$authorizedRequest,
					'GET',
					token,
					'/api/users/' + (userId + '/work'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$UserWorkReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
			case 'UserSubmissionsPage':
				var userId = page.a;
				return A3($author$project$Sharecrop$Api$fetchUserSubmissionsPage, token, userId, 0);
			case 'CollectibleDetailPage':
				return $author$project$Sharecrop$Api$fetchCollectibles(token);
			case 'SeriesListPage':
				return $author$project$Sharecrop$Api$fetchSeriesList(token);
			case 'SeriesDetailPage':
				var seriesId = page.a;
				return A2($author$project$Sharecrop$Api$fetchSeriesDetail, token, seriesId);
			case 'TeamDetailPage':
				var teamId = page.a;
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							A5(
							$author$project$Sharecrop$Api$authorizedRequest,
							'GET',
							token,
							'/api/teams/' + teamId,
							$elm$http$Http$emptyBody,
							A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TeamDetailReceived, $author$project$Sharecrop$Generated$Team$teamDetailResponseDecoder)),
							A6($author$project$Sharecrop$Api$fetchTeamWork, token, teamId, '', '', 'newest', 0),
							A2($author$project$Sharecrop$Api$fetchTeamCollectibles, token, teamId)
						]));
			case 'AdminPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							A5(
							$author$project$Sharecrop$Api$authorizedRequest,
							'GET',
							token,
							'/api/admin/operations',
							$elm$http$Http$emptyBody,
							A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OperationsReceived, $author$project$Sharecrop$Generated$Admin$operationsResponseDecoder)),
							A5($author$project$Sharecrop$Api$fetchAuditEvents, token, '', '', '', 0),
							A2($author$project$Sharecrop$Api$fetchPlatformAdmins, token, 0),
							$author$project$Sharecrop$Api$fetchUserDirectory(token),
							A3($author$project$Sharecrop$Api$fetchAdminModerationReports, token, 'open', 0),
							A2($author$project$Sharecrop$Api$fetchAdminPrivacyRequests, token, 0)
						]));
			case 'InboxPage':
				return A2($author$project$Sharecrop$Api$fetchNotifications, token, 0);
			default:
				return $elm$core$Platform$Cmd$none;
		}
	});
var $author$project$Sharecrop$Types$PrivacyRetentionRunReceived = function (a) {
	return {$: 'PrivacyRetentionRunReceived', a: a};
};
var $author$project$Sharecrop$Generated$Privacy$PrivacyRetentionRunResponse = function (redactedFieldCount) {
	return {redactedFieldCount: redactedFieldCount};
};
var $author$project$Sharecrop$Generated$Privacy$privacyRetentionRunResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Privacy$PrivacyRetentionRunResponse,
	A2($elm$json$Json$Decode$field, 'redacted_field_count', $elm$json$Json$Decode$int));
var $author$project$Sharecrop$Api$runPrivacyRetention = function (token) {
	return A5(
		$author$project$Sharecrop$Api$authorizedRequest,
		'POST',
		token,
		'/api/admin/privacy-retention/run',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$PrivacyRetentionRunReceived, $author$project$Sharecrop$Generated$Privacy$privacyRetentionRunResponseDecoder));
};
var $author$project$Main$saveQueueView = F2(
	function (view, views) {
		return A2(
			$elm$core$List$cons,
			view,
			A2(
				$elm$core$List$filter,
				function (existing) {
					return !_Utils_eq(existing.name, view.name);
				},
				views));
	});
var $author$project$Sharecrop$Types$SavedQueueViewSaved = function (a) {
	return {$: 'SavedQueueViewSaved', a: a};
};
var $author$project$Sharecrop$Api$saveSavedQueueView = F3(
	function (token, scope, view) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/saved-queue-views',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'scope',
							$elm$json$Json$Encode$string(scope)),
							_Utils_Tuple2(
							'name',
							$elm$json$Json$Encode$string(view.name)),
							_Utils_Tuple2(
							'query',
							$elm$json$Json$Encode$string(view.query)),
							_Utils_Tuple2(
							'state_filter',
							$elm$json$Json$Encode$string(view.stateFilter)),
							_Utils_Tuple2(
							'type_filter',
							$elm$json$Json$Encode$string(view.typeFilter)),
							_Utils_Tuple2(
							'sort',
							$elm$json$Json$Encode$string(view.sort))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SavedQueueViewSaved, $author$project$Sharecrop$Generated$SavedQueueViews$savedQueueViewResponseDecoder));
	});
var $elm$file$File$Select$file = F2(
	function (mimes, toMsg) {
		return A2(
			$elm$core$Task$perform,
			toMsg,
			_File_uploadOne(mimes));
	});
var $author$project$Main$selectAttachment = function (toMsg) {
	return A2($elm$file$File$Select$file, $author$project$Main$allowedAttachmentTypes, toMsg);
};
var $author$project$Sharecrop$Api$seriesFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.series;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$seriesListRefresh = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return _Utils_eq(state.page, $author$project$Sharecrop$Types$SeriesListPage) ? $author$project$Sharecrop$Api$fetchSeriesList(state.accessToken) : $elm$core$Platform$Cmd$none;
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Sharecrop$Api$indexOf = F2(
	function (value, items) {
		return A2(
			$elm$core$Maybe$map,
			$elm$core$Tuple$first,
			$elm$core$List$head(
				A2(
					$elm$core$List$filter,
					function (_v0) {
						var item = _v0.b;
						return _Utils_eq(item, value);
					},
					A2(
						$elm$core$List$indexedMap,
						F2(
							function (index, item) {
								return _Utils_Tuple2(index, item);
							}),
						items))));
	});
var $elm$core$List$drop = F2(
	function (n, list) {
		drop:
		while (true) {
			if (n <= 0) {
				return list;
			} else {
				if (!list.b) {
					return list;
				} else {
					var x = list.a;
					var xs = list.b;
					var $temp$n = n - 1,
						$temp$list = xs;
					n = $temp$n;
					list = $temp$list;
					continue drop;
				}
			}
		}
	});
var $author$project$Sharecrop$Api$swapAt = F3(
	function (a, b, items) {
		var valueAt = function (index) {
			return $elm$core$List$head(
				A2($elm$core$List$drop, index, items));
		};
		var _v0 = _Utils_Tuple2(
			valueAt(a),
			valueAt(b));
		if ((_v0.a.$ === 'Just') && (_v0.b.$ === 'Just')) {
			var va = _v0.a.a;
			var vb = _v0.b.a;
			return A2(
				$elm$core$List$indexedMap,
				F2(
					function (index, item) {
						return _Utils_eq(index, a) ? vb : (_Utils_eq(index, b) ? va : item);
					}),
				items);
		} else {
			return items;
		}
	});
var $author$project$Sharecrop$Api$moveSeriesTaskOrder = F3(
	function (up, taskId, tasks) {
		var ids = A2(
			$elm$core$List$map,
			function ($) {
				return $.id;
			},
			tasks);
		var _v0 = A2($author$project$Sharecrop$Api$indexOf, taskId, ids);
		if (_v0.$ === 'Just') {
			var index = _v0.a;
			var target = up ? (index - 1) : (index + 1);
			return ((target < 0) || (_Utils_cmp(
				target,
				$elm$core$List$length(ids)) > -1)) ? ids : A3($author$project$Sharecrop$Api$swapAt, index, target, ids);
		} else {
			return ids;
		}
	});
var $author$project$Sharecrop$Api$reorderSeriesCommand = F3(
	function (token, seriesId, taskIds) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/task-series/' + (seriesId + '/reorder'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'task_ids',
							A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, taskIds))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesMutationReceived, $author$project$Sharecrop$Api$seriesDetailDecoder));
	});
var $author$project$Sharecrop$Api$withSession = F2(
	function (model, run) {
		var _v0 = model.session;
		if (_v0.$ === 'LoggedIn') {
			var state = _v0.a;
			return run(state);
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Main$seriesReorder = F4(
	function (model, seriesId, taskId, up) {
		return A2(
			$author$project$Sharecrop$Api$withSession,
			model,
			function (state) {
				var _v0 = state.seriesDetail;
				if (_v0.$ === 'Just') {
					var data = _v0.a;
					return _Utils_Tuple2(
						model,
						A3(
							$author$project$Sharecrop$Api$reorderSeriesCommand,
							state.accessToken,
							seriesId,
							A3($author$project$Sharecrop$Api$moveSeriesTaskOrder, up, taskId, data.tasks)));
				} else {
					return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
				}
			});
	});
var $author$project$Sharecrop$Api$seriesStateCommand = F3(
	function (token, seriesId, action) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/task-series/' + (seriesId + ('/' + action)),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesMutationReceived, $author$project$Sharecrop$Api$seriesDetailDecoder));
	});
var $author$project$Sharecrop$Api$submissionsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.submissions;
	} else {
		return _List_Nil;
	}
};
var $author$project$Sharecrop$Types$SubmitReceived = function (a) {
	return {$: 'SubmitReceived', a: a};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionCreatedResponse = F2(
	function (submission, receiptToken) {
		return {receiptToken: receiptToken, submission: submission};
	});
var $author$project$Sharecrop$Generated$Submission$submissionCreatedResponseDecoder = A3(
	$elm$json$Json$Decode$map2,
	$author$project$Sharecrop$Generated$Submission$SubmissionCreatedResponse,
	A2($elm$json$Json$Decode$field, 'submission', $author$project$Sharecrop$Generated$Submission$submissionResponseDecoder),
	A2($elm$json$Json$Decode$field, 'receipt_token', $elm$json$Json$Decode$string));
var $author$project$Sharecrop$Api$submissionRequestBody = F2(
	function (responseJson, attachments) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'response_json',
					$elm$json$Json$Encode$string(responseJson)),
					_Utils_Tuple2(
					'attachments',
					A2($elm$json$Json$Encode$list, $author$project$Sharecrop$Api$attachmentRequestBody, attachments))
				]));
	});
var $author$project$Sharecrop$Api$postSubmission = F4(
	function (token, taskId, responseJson, attachments) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/submissions'),
			$elm$http$Http$jsonBody(
				A2($author$project$Sharecrop$Api$submissionRequestBody, responseJson, attachments)),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SubmitReceived, $author$project$Sharecrop$Generated$Submission$submissionCreatedResponseDecoder));
	});
var $elm$json$Json$Decode$value = _Json_decodeValue;
var $author$project$Sharecrop$Api$submitCommand = F2(
	function (model, state) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			var trimmed = $elm$core$String$trim(state.submitInput);
			if (trimmed === '') {
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (current) {
							return _Utils_update(
								current,
								{
									submitMessage: $elm$core$Maybe$Just('Enter a response first.')
								});
						}),
					$elm$core$Platform$Cmd$none);
			} else {
				var _v1 = A2($elm$json$Json$Decode$decodeString, $elm$json$Json$Decode$value, trimmed);
				if (_v1.$ === 'Ok') {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (current) {
								return _Utils_update(
									current,
									{submitMessage: $elm$core$Maybe$Nothing});
							}),
						A4($author$project$Sharecrop$Api$postSubmission, state.accessToken, taskId, trimmed, state.submitAttachments));
				} else {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (current) {
								return _Utils_update(
									current,
									{
										submitMessage: $elm$core$Maybe$Just('Response must be valid JSON.')
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			}
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Sharecrop$Labels$submissionStateLabel = function (state) {
	switch (state.$) {
		case 'SubmissionStateSubmitted':
			return 'submitted';
		case 'SubmissionStateInvalid':
			return 'invalid';
		case 'SubmissionStateAccepted':
			return 'accepted';
		case 'SubmissionStateRejected':
			return 'rejected';
		default:
			return 'changes requested';
	}
};
var $author$project$Sharecrop$View$submitSuccessLabel = function (created) {
	var base = 'Submission ' + (created.submission.id + (' (' + ($author$project$Sharecrop$Labels$submissionStateLabel(created.submission.state) + ').')));
	return $elm$core$List$isEmpty(created.submission.validationErrors) ? base : (base + (' ' + A2(
		$elm$core$String$join,
		'; ',
		A2(
			$elm$core$List$map,
			function (error) {
				return error.path + (': ' + error.message);
			},
			created.submission.validationErrors))));
};
var $author$project$Sharecrop$View$taskTemplate = function (taskType) {
	switch (taskType) {
		case 'code_review':
			return $elm$core$Maybe$Just(
				{description: 'Review the linked pull request. Identify correctness, design, and style issues, then give an overall verdict.', schema: '{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"issues\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"verdict\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"approve\",\"request_changes\",\"comment\"]}}]}'});
		case 'security_review':
			return $elm$core$Maybe$Just(
				{description: 'Perform a security review of the linked code. List vulnerabilities with remediation and an overall severity.', schema: '{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"findings\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"severity\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"none\",\"low\",\"medium\",\"high\",\"critical\"]}}]}'});
		case 'product_review':
			return $elm$core$Maybe$Just(
				{description: 'Review the linked product or feature. Assess clarity, value, and gaps, then recommend next steps.', schema: '{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"strengths\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"recommendations\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}}]}'});
		case 'ui_ux_review':
			return $elm$core$Maybe$Just(
				{description: 'Review the linked UI/UX. Check usability, accessibility, and visual consistency, then list issues.', schema: '{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"issues\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"accessibility\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"pass\",\"fail\"]}}]}'});
		case 'qa_testing':
			return $elm$core$Maybe$Just(
				{description: 'Test the linked build against its requirements. Report the cases you ran and the overall result.', schema: '{\"kind\":\"object\",\"fields\":[{\"name\":\"summary\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"}},{\"name\":\"cases\",\"presence\":\"required\",\"schema\":{\"kind\":\"array\",\"item\":{\"kind\":\"string\"}}},{\"name\":\"result\",\"presence\":\"required\",\"schema\":{\"kind\":\"enum\",\"values\":[\"pass\",\"fail\"]}}]}'});
		default:
			return $elm$core$Maybe$Nothing;
	}
};
var $author$project$Sharecrop$Api$tasksFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.tasks;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$teamWorkSavedViewScope = 'team_work';
var $author$project$Sharecrop$Api$teamsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.teams;
	} else {
		return _List_Nil;
	}
};
var $elm$url$Url$addPort = F2(
	function (maybePort, starter) {
		if (maybePort.$ === 'Nothing') {
			return starter;
		} else {
			var port_ = maybePort.a;
			return starter + (':' + $elm$core$String$fromInt(port_));
		}
	});
var $elm$url$Url$addPrefixed = F3(
	function (prefix, maybeSegment, starter) {
		if (maybeSegment.$ === 'Nothing') {
			return starter;
		} else {
			var segment = maybeSegment.a;
			return _Utils_ap(
				starter,
				_Utils_ap(prefix, segment));
		}
	});
var $elm$url$Url$toString = function (url) {
	var http = function () {
		var _v0 = url.protocol;
		if (_v0.$ === 'Http') {
			return 'http://';
		} else {
			return 'https://';
		}
	}();
	return A3(
		$elm$url$Url$addPrefixed,
		'#',
		url.fragment,
		A3(
			$elm$url$Url$addPrefixed,
			'?',
			url.query,
			_Utils_ap(
				A2(
					$elm$url$Url$addPort,
					url.port_,
					_Utils_ap(http, url.host)),
				url.path)));
};
var $author$project$Sharecrop$Api$toggleScope = F2(
	function (scope, scopes) {
		return A2($elm$core$List$member, scope, scopes) ? A2(
			$elm$core$List$filter,
			function (existing) {
				return !_Utils_eq(existing, scope);
			},
			scopes) : A2($elm$core$List$cons, scope, scopes);
	});
var $author$project$Main$toggleString = F2(
	function (value, values) {
		return A2($elm$core$List$member, value, values) ? A2(
			$elm$core$List$filter,
			function (existing) {
				return !_Utils_eq(existing, value);
			},
			values) : A2($elm$core$List$cons, value, values);
	});
var $author$project$Sharecrop$Types$TransferCollectibleReceived = function (a) {
	return {$: 'TransferCollectibleReceived', a: a};
};
var $author$project$Sharecrop$Api$transferCollectible = F3(
	function (token, collectibleId, recipientId) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/collectibles/' + (collectibleId + '/transfer'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'recipient_id',
							$elm$json$Json$Encode$string(recipientId))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$TransferCollectibleReceived, $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder));
	});
var $author$project$Sharecrop$Types$AdminModerationReportTriaged = function (a) {
	return {$: 'AdminModerationReportTriaged', a: a};
};
var $author$project$Sharecrop$Api$triageModerationReport = F4(
	function (token, reportID, stateValue, resolutionNote) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'POST',
			token,
			'/api/admin/moderation/reports/' + (reportID + '/triage'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'state',
							$elm$json$Json$Encode$string(stateValue)),
							_Utils_Tuple2(
							'resolution_note',
							$elm$json$Json$Encode$string(resolutionNote))
						]))),
			A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$AdminModerationReportTriaged, $author$project$Sharecrop$Generated$Moderation$moderationReportResponseDecoder));
	});
var $author$project$Main$updateFieldAt = F3(
	function (index, transform, fields) {
		return A2(
			$elm$core$List$indexedMap,
			F2(
				function (i, field) {
					return _Utils_eq(i, index) ? transform(field) : field;
				}),
			fields);
	});
var $author$project$Sharecrop$Types$UpdateMemberRolesReceived = function (a) {
	return {$: 'UpdateMemberRolesReceived', a: a};
};
var $author$project$Sharecrop$Api$updateMemberRolesCommand = F4(
	function (model, state, userId, roles) {
		return ((state.activeOrgId === '') || $elm$core$List$isEmpty(roles)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							provisionMemberMessage: $elm$core$Maybe$Just('Select at least one role.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{provisionMemberMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'PATCH',
				state.accessToken,
				'/api/organizations/' + (state.activeOrgId + ('/members/' + (userId + '/roles'))),
				$elm$http$Http$jsonBody(
					$elm$json$Json$Encode$object(
						_List_fromArray(
							[
								_Utils_Tuple2(
								'roles',
								A2($elm$json$Json$Encode$list, $elm$json$Json$Encode$string, roles))
							]))),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$UpdateMemberRolesReceived, $author$project$Sharecrop$Generated$Organization$organizationMemberResponseDecoder)));
	});
var $author$project$Sharecrop$Api$updateProfile = F2(
	function (token, email) {
		return A5(
			$author$project$Sharecrop$Api$authorizedRequest,
			'PATCH',
			token,
			'/api/account/profile',
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'email',
							$elm$json$Json$Encode$string(email))
						]))),
			$elm$http$Http$expectWhatever($author$project$Sharecrop$Types$AccountActionReceived));
	});
var $author$project$Sharecrop$Api$updateSeriesCommand = F3(
	function (model, state, seriesId) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.seriesRenameTitle)) ? _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							seriesMessage: $elm$core$Maybe$Just('A series title is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Sharecrop$Api$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{seriesMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Sharecrop$Api$authorizedRequest,
				'PATCH',
				state.accessToken,
				'/api/task-series/' + seriesId,
				$elm$http$Http$jsonBody(
					A2($author$project$Sharecrop$Api$seriesBody, state.seriesRenameTitle, state.seriesRenameDescription)),
				A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$SeriesMutationReceived, $author$project$Sharecrop$Api$seriesDetailDecoder)));
	});
var $author$project$Main$update = F2(
	function (msg, model) {
		switch (msg.$) {
			case 'EmailChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{email: value}),
					$elm$core$Platform$Cmd$none);
			case 'PasswordChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{password: value}),
					$elm$core$Platform$Cmd$none);
			case 'RegisterClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{authError: $elm$core$Maybe$Nothing}),
					A2($author$project$Sharecrop$Api$postAuth, '/api/auth/register', model));
			case 'LoginClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{authError: $elm$core$Maybe$Nothing}),
					A2($author$project$Sharecrop$Api$postAuth, '/api/auth/login', model));
			case 'GuestClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{authError: $elm$core$Maybe$Nothing}),
					$author$project$Sharecrop$Api$postGuest);
			case 'AuthReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					var state = A2($author$project$Main$loggedInForPage, response, model.route);
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Nothing,
								password: '',
								session: $author$project$Sharecrop$Types$LoggedIn(
									_Utils_update(
										state,
										{accountEmail: model.email}))
							}),
						$author$project$Sharecrop$Api$loadAfterAuth(response.accessToken));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just(
									$author$project$Sharecrop$Labels$httpErrorLabel(error))
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'RefreshReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								session: $author$project$Sharecrop$Types$LoggedIn(
									A2($author$project$Main$loggedInForPage, response, model.route))
							}),
						$elm$core$Platform$Cmd$batch(
							_List_fromArray(
								[
									$author$project$Sharecrop$Api$loadAfterAuth(response.accessToken),
									A3($author$project$Sharecrop$Api$routeLoadCmd, response.accessToken, response.subjectID, model.route)
								])));
				} else {
					return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
				}
			case 'PasswordResetEmailChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{resetEmail: value}),
					$elm$core$Platform$Cmd$none);
			case 'PasswordResetTokenChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{resetToken: value}),
					$elm$core$Platform$Cmd$none);
			case 'PasswordResetPasswordChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{resetPassword: value}),
					$elm$core$Platform$Cmd$none);
			case 'RequestPasswordResetClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{authError: $elm$core$Maybe$Nothing}),
					$author$project$Sharecrop$Api$requestPasswordReset(model));
			case 'ConfirmPasswordResetClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{authError: $elm$core$Maybe$Nothing}),
					$author$project$Sharecrop$Api$confirmPasswordReset(model));
			case 'PasswordResetRequested':
				if (msg.a.$ === 'Ok') {
					var token = msg.a.a;
					return (token === '') ? _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just('Password reset instructions sent.')
							}),
						$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just('Password reset token created.'),
								resetToken: token
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just(
									$author$project$Sharecrop$Labels$httpErrorLabel(error))
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'PasswordResetConfirmed':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just('Password reset. Log in with the new password.'),
								resetPassword: '',
								resetToken: ''
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just(
									$author$project$Sharecrop$Labels$httpErrorLabel(error))
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'BalanceReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									balance: $author$project$Sharecrop$Api$balanceFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'LedgerReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									entries: $author$project$Sharecrop$Api$entriesFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'PreviousLedgerPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.ledgerOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{ledgerOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchLedger, state.accessToken, offset));
					});
			case 'NextLedgerPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.ledgerOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{ledgerOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchLedger, state.accessToken, offset));
					});
			case 'TasksReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									tasks: $author$project$Sharecrop$Api$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TaskStateFilterChanged':
				var value = msg.a;
				var updated = A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{taskListOffset: 0, taskStateFilter: value});
					});
				return A2(
					$author$project$Sharecrop$Api$withSession,
					updated,
					function (state) {
						return _Utils_Tuple2(
							updated,
							A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, value, state.taskListTypeFilter, state.taskListSort, 0));
					});
			case 'TaskListQueryChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{taskListQuery: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TaskListTypeFilterChanged':
				var value = msg.a;
				var updated = A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{taskListOffset: 0, taskListTypeFilter: value});
					});
				return A2(
					$author$project$Sharecrop$Api$withSession,
					updated,
					function (state) {
						return _Utils_Tuple2(
							updated,
							A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, value, state.taskListSort, 0));
					});
			case 'TaskListSortChanged':
				var value = msg.a;
				var updated = A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{taskListOffset: 0, taskListSort: value});
					});
				return A2(
					$author$project$Sharecrop$Api$withSession,
					updated,
					function (state) {
						return _Utils_Tuple2(
							updated,
							A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, state.taskListTypeFilter, value, 0));
					});
			case 'PreviousTasksPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.taskListOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{taskListOffset: offset});
								}),
							A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, state.taskListTypeFilter, state.taskListSort, offset));
					});
			case 'NextTasksPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.taskListOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{taskListOffset: offset});
								}),
							A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, state.taskListTypeFilter, state.taskListSort, offset));
					});
			case 'CreateTitleChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createTitle: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateDescriptionChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createDescription: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateResponseSchemaChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createResponseSchema: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AddSchemaFieldClicked':
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							function (fields) {
								return _Utils_ap(
									fields,
									_List_fromArray(
										[
											{enumValues: '', itemKind: 'string', kind: 'string', name: '', required: true}
										]));
							})),
					$elm$core$Platform$Cmd$none);
			case 'RemoveSchemaFieldClicked':
				var index = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							function (fields) {
								return A2(
									$elm$core$List$map,
									$elm$core$Tuple$second,
									A2(
										$elm$core$List$filter,
										function (_v1) {
											var i = _v1.a;
											return !_Utils_eq(i, index);
										},
										A2($elm$core$List$indexedMap, $elm$core$Tuple$pair, fields)));
							})),
					$elm$core$Platform$Cmd$none);
			case 'SchemaFieldNameChanged':
				var index = msg.a;
				var value = msg.b;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							A2(
								$author$project$Main$updateFieldAt,
								index,
								function (field) {
									return _Utils_update(
										field,
										{name: value});
								}))),
					$elm$core$Platform$Cmd$none);
			case 'SchemaFieldKindChanged':
				var index = msg.a;
				var value = msg.b;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							A2(
								$author$project$Main$updateFieldAt,
								index,
								function (field) {
									return _Utils_update(
										field,
										{kind: value});
								}))),
					$elm$core$Platform$Cmd$none);
			case 'SchemaFieldRequiredChanged':
				var index = msg.a;
				var value = msg.b;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							A2(
								$author$project$Main$updateFieldAt,
								index,
								function (field) {
									return _Utils_update(
										field,
										{required: value});
								}))),
					$elm$core$Platform$Cmd$none);
			case 'SchemaFieldItemKindChanged':
				var index = msg.a;
				var value = msg.b;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							A2(
								$author$project$Main$updateFieldAt,
								index,
								function (field) {
									return _Utils_update(
										field,
										{itemKind: value});
								}))),
					$elm$core$Platform$Cmd$none);
			case 'SchemaFieldEnumValuesChanged':
				var index = msg.a;
				var value = msg.b;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						$author$project$Main$applySchemaFields(
							A2(
								$author$project$Main$updateFieldAt,
								index,
								function (field) {
									return _Utils_update(
										field,
										{enumValues: value});
								}))),
					$elm$core$Platform$Cmd$none);
			case 'CreatePayloadChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createPayloadJson: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateRewardKindChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createRewardCollectibleIds: _List_Nil, createRewardKind: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateRewardAmountChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createRewardAmount: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ToggleCreateRewardCollectible':
				var collectibleId = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									createRewardCollectibleIds: A2($author$project$Main$toggleString, collectibleId, state.createRewardCollectibleIds)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateVisibilityChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createScopeOrganizationId: '', createScopeTeamId: '', createScopeUserId: '', createVisibility: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateScopeUserIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createScopeUserId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateScopeTeamIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createScopeTeamId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateScopeOrganizationIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createScopeOrganizationId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateAssigneeScopeChosen':
				var scope = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createAssigneeScope: scope});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateParticipationChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createParticipationPolicy: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateReservationHoursChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createReservationHours: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'PickCreateAttachmentClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return (_Utils_cmp(
							$elm$core$List$length(state.createAttachments),
							$author$project$Main$attachmentMaxCount) > -1) ? _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											createMessage: $elm$core$Maybe$Just('Attach up to 5 files.')
										});
								}),
							$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
							model,
							$author$project$Main$selectAttachment($author$project$Sharecrop$Types$CreateAttachmentFileChosen));
					});
			case 'CreateAttachmentFileChosen':
				var file = msg.a;
				return _Utils_Tuple2(
					model,
					$author$project$Main$readCreateAttachment(file));
			case 'CreateAttachmentSelected':
				var name = msg.a;
				var contentType = msg.b;
				var sizeBytes = msg.c;
				var dataURL = msg.d;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									createAttachments: _Utils_ap(
										state.createAttachments,
										_List_fromArray(
											[
												{contentType: contentType, dataURL: dataURL, name: name, sizeBytes: sizeBytes}
											])),
									createMessage: $elm$core$Maybe$Nothing
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateAttachmentRejected':
				var message = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									createMessage: $elm$core$Maybe$Just(message)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'RemoveCreateAttachmentClicked':
				var index = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									createAttachments: A2($author$project$Main$removeAt, index, state.createAttachments)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateTaskClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$createTaskCommand, model, state);
					});
			case 'CreateTaskReceived':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createAttachments: _List_Nil,
										createDescription: '',
										createMessage: $elm$core$Maybe$Just('Created task ' + created.id),
										createParticipationPolicy: $author$project$Sharecrop$Labels$participationPolicyTag($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen),
										createPayloadJson: '',
										createReferenceURL: '',
										createReservationHours: '48',
										createResponseSchema: '{\"kind\":\"freeform\"}',
										createRewardCollectibleIds: _List_Nil,
										createSchemaFields: _List_Nil,
										createTaskType: 'general',
										createTitle: '',
										fundAmount: (created.rewardKind === 'credit') ? $elm$core$String$fromInt(created.rewardCreditAmount) : state.fundAmount,
										fundTaskId: created.id
									});
							}),
						$author$project$Sharecrop$Api$refreshTasksAndLedger(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CredentialsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									credentials: $author$project$Sharecrop$Api$credentialsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'FundTaskIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{fundTaskId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'FundAmountChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{fundAmount: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'FundOrganizationIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{fundOrganizationId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'FundClicked':
				var bumped = A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{fundNonce: state.fundNonce + 1});
					});
				return A2(
					$author$project$Sharecrop$Api$withSession,
					bumped,
					function (state) {
						return A2($author$project$Sharecrop$Api$fundTaskCommand, bumped, state);
					});
			case 'FundReceived':
				if (msg.a.$ === 'Ok') {
					var escrow = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										fundMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$View$fundSuccessLabel(escrow))
									});
							}),
						$author$project$Sharecrop$Api$refreshLedger(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										fundMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OpenTaskClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Sharecrop$Api$postOpenTask, state.accessToken, taskId));
					});
			case 'OpenTaskReceived':
				if (msg.a.$ === 'Ok') {
					var detail = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										detail: $elm$core$Maybe$Just(detail),
										taskActionMessage: $elm$core$Maybe$Just('Task opened.')
									});
							}),
						$author$project$Sharecrop$Api$refreshTasksAndDiscovery(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'RefundTaskClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Sharecrop$Api$postRefundTask, state.accessToken, taskId));
					});
			case 'RefundTaskReceived':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just('Task refunded and cancelled.')
									});
							}),
						$elm$core$Platform$Cmd$batch(
							_List_fromArray(
								[
									$author$project$Sharecrop$Api$refreshTasksAndLedger(model),
									$author$project$Sharecrop$Api$refreshAfterAccept(model)
								])));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CancelTaskClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Sharecrop$Api$postCancelTask, state.accessToken, taskId));
					});
			case 'CancelTaskReceived':
				if (msg.a.$ === 'Ok') {
					var detail = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										detail: $elm$core$Maybe$Just(detail),
										taskActionMessage: $elm$core$Maybe$Just('Task cancelled.')
									});
							}),
						$author$project$Sharecrop$Api$refreshTasksAndDiscovery(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'RefundCollectibleRewardClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Sharecrop$Api$postRefundCollectibleReward, state.accessToken, taskId));
					});
			case 'RefundCollectibleRewardReceived':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just('Collectible reward refunded.')
									});
							}),
						$elm$core$Platform$Cmd$batch(
							_List_fromArray(
								[
									$author$project$Sharecrop$Api$refreshAfterAccept(model),
									$author$project$Sharecrop$Api$refreshCollectibles(model)
								])));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AgentLabelChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{agentLabel: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ToggleScope':
				var scope = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									agentScopes: A2($author$project$Sharecrop$Api$toggleScope, scope, state.agentScopes)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateAgentClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$createAgentCommand, model, state);
					});
			case 'AgentCreated':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										agentMessage: $elm$core$Maybe$Nothing,
										newCredential: $elm$core$Maybe$Just(created)
									});
							}),
						$author$project$Sharecrop$Api$refreshCredentials(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										agentMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ToggleTaskIntegration':
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{taskIntegrationOpen: !state.taskIntegrationOpen});
						}),
					$elm$core$Platform$Cmd$none);
			case 'MintTaskTokenClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							$author$project$Sharecrop$Api$mintTaskToken(state.accessToken));
					});
			case 'TaskTokenMinted':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskAgentToken: $elm$core$Maybe$Just(created.secret)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just(
											'Could not create agent token: ' + $author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'MintUserTokenClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							$author$project$Sharecrop$Api$mintUserToken(state.accessToken));
					});
			case 'UserTokenMinted':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										userAgentToken: $elm$core$Maybe$Just(created.secret)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskActionMessage: $elm$core$Maybe$Just(
											'Could not create agent token: ' + $author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CopyClicked':
				var clipboardText = msg.a;
				return _Utils_Tuple2(
					model,
					$author$project$Main$copyToClipboard(clipboardText));
			case 'RevokeClicked':
				var credentialId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Sharecrop$Api$revokeAgent, state.accessToken, credentialId));
					});
			case 'AgentRevoked':
				return _Utils_Tuple2(
					model,
					$author$project$Sharecrop$Api$refreshCredentials(model));
			case 'LogoutClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{email: '', password: '', session: $author$project$Sharecrop$Types$LoggedOut}),
					$elm$core$Platform$Cmd$batch(
						_List_fromArray(
							[
								$author$project$Sharecrop$Api$postLogout,
								A2($elm$browser$Browser$Navigation$pushUrl, model.key, '#/')
							])));
			case 'LogoutReceived':
				return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
			case 'DiscoveryIncludeReservedChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var nextState = _Utils_update(
							state,
							{discoveryIncludeReserved: value, discoveryOffset: 0});
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (_v2) {
									return nextState;
								}),
							A3($author$project$Sharecrop$Api$fetchDiscovery, state.accessToken, value, 0));
					});
			case 'DiscoveryQueryChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{discoveryQuery: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'PreviousDiscoveryPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.discoveryOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{discoveryOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchDiscovery, state.accessToken, state.discoveryIncludeReserved, offset));
					});
			case 'NextDiscoveryPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.discoveryOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{discoveryOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchDiscovery, state.accessToken, state.discoveryIncludeReserved, offset));
					});
			case 'DiscoveryReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									discoveryTasks: $author$project$Sharecrop$Api$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'DiscoveryViewClicked':
				var taskId = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (s) {
							return _Utils_update(
								s,
								{activeSubmissionCommentsID: $elm$core$Maybe$Nothing, detail: $elm$core$Maybe$Nothing, detailError: $elm$core$Maybe$Nothing, reservationMessage: $elm$core$Maybe$Nothing, reservations: _List_Nil, reviewBan: false, reviewMessage: $elm$core$Maybe$Nothing, reviewNote: '', reviewPartialCredit: '', reviewTip: '', reviewTipCollectibleId: '', submissionCommentBody: '', submissions: _List_Nil, submitInput: '', submitMessage: $elm$core$Maybe$Nothing, taskActionMessage: $elm$core$Maybe$Nothing, taskAgentToken: $elm$core$Maybe$Nothing, taskCommentBody: '', taskComments: _List_Nil, taskIntegrationOpen: false});
						}),
					A2($elm$browser$Browser$Navigation$pushUrl, model.key, '#/tasks/' + taskId));
			case 'DetailReceived':
				if (msg.a.$ === 'Ok') {
					var detail = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										detail: $elm$core$Maybe$Just(detail),
										detailError: $elm$core$Maybe$Nothing
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										detailError: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ReserveClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{reservationMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Sharecrop$Api$postReservation, state, taskId));
					});
			case 'ReservationOrganizationIdChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTeamOffset: 0, orgTeamQuery: '', orgTeams: _List_Nil, reservationOrganizationId: value, reservationTeamId: ''});
								}),
							(value === '') ? $elm$core$Platform$Cmd$none : A2($author$project$Sharecrop$Api$fetchOrgTeams, state.accessToken, value));
					});
			case 'ReservationTeamIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reservationTeamId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReservationReceived':
				if (msg.a.$ === 'Ok') {
					var reservation = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$View$reservationSuccessLabel(reservation))
									});
							}),
						$author$project$Sharecrop$Api$refreshDetailReservations(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ReservationsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{reservations: response.reservations});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{reservations: _List_Nil});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ApproveReservationClicked':
				var reservationId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A4($author$project$Sharecrop$Api$reservationChangeCommand, model, state, reservationId, 'approve');
					});
			case 'DeclineReservationClicked':
				var reservationId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A4($author$project$Sharecrop$Api$reservationChangeCommand, model, state, reservationId, 'decline');
					});
			case 'CancelReservationClicked':
				var reservationId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A4($author$project$Sharecrop$Api$reservationChangeCommand, model, state, reservationId, 'cancel');
					});
			case 'ReservationChangeReceived':
				if (msg.a.$ === 'Ok') {
					var reservation = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$View$reservationSuccessLabel(reservation))
									});
							}),
						$author$project$Sharecrop$Api$refreshDetailReservations(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SubmissionsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{submissions: response.submissions});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reviewMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										submissions: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SubmitInputChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{submitInput: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'PickSubmitAttachmentClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return (_Utils_cmp(
							$elm$core$List$length(state.submitAttachments),
							$author$project$Main$attachmentMaxCount) > -1) ? _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											submitMessage: $elm$core$Maybe$Just('Attach up to 5 files.')
										});
								}),
							$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
							model,
							$author$project$Main$selectAttachment($author$project$Sharecrop$Types$SubmitAttachmentFileChosen));
					});
			case 'SubmitAttachmentFileChosen':
				var file = msg.a;
				return _Utils_Tuple2(
					model,
					$author$project$Main$readSubmitAttachment(file));
			case 'SubmitAttachmentSelected':
				var name = msg.a;
				var contentType = msg.b;
				var sizeBytes = msg.c;
				var dataURL = msg.d;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									submitAttachments: _Utils_ap(
										state.submitAttachments,
										_List_fromArray(
											[
												{contentType: contentType, dataURL: dataURL, name: name, sizeBytes: sizeBytes}
											])),
									submitMessage: $elm$core$Maybe$Nothing
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SubmitAttachmentRejected':
				var message = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									submitMessage: $elm$core$Maybe$Just(message)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'RemoveSubmitAttachmentClicked':
				var index = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									submitAttachments: A2($author$project$Main$removeAt, index, state.submitAttachments)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SubmitClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$submitCommand, model, state);
					});
			case 'SubmitReceived':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return A2(
						$author$project$Sharecrop$Api$withSession,
						model,
						function (state) {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												activeSubmissionCommentsID: $elm$core$Maybe$Just(created.submission.id),
												submissionCommentMessage: $elm$core$Maybe$Nothing,
												submissionComments: _List_Nil,
												submitAttachments: _List_Nil,
												submitInput: '',
												submitMessage: $elm$core$Maybe$Just(
													$author$project$Sharecrop$View$submitSuccessLabel(created))
											});
									}),
								$elm$core$Platform$Cmd$batch(
									_List_fromArray(
										[
											$author$project$Sharecrop$Api$refreshDetailSubmissions(model),
											A2($author$project$Sharecrop$Api$fetchSubmissionComments, state.accessToken, created.submission.id)
										])));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submitMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ModerationReasonChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{moderationReason: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ModerationDetailsChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{moderationDetails: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReportTaskClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{moderationMessage: $elm$core$Maybe$Nothing});
								}),
							A4($author$project$Sharecrop$Api$reportTask, state.accessToken, taskId, state.moderationReason, state.moderationDetails));
					});
			case 'ModerationReportReceived':
				if (msg.a.$ === 'Ok') {
					var report = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										moderationDetails: '',
										moderationMessage: $elm$core$Maybe$Just('Report submitted: ' + report.reason)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										moderationMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ReviewNoteChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reviewNote: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReviewPartialCreditChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reviewPartialCredit: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReviewTipChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reviewTip: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReviewTipCollectibleChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reviewTipCollectibleId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReviewBanChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reviewBan: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AcceptClicked':
				var submissionId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$acceptCommand, model, state, submissionId);
					});
			case 'RequestChangesClicked':
				var submissionId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$requestChangesCommand, model, state, submissionId);
					});
			case 'RejectClicked':
				var submissionId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$rejectCommand, model, state, submissionId);
					});
			case 'ReviewActionReceived':
				if (msg.b.$ === 'Ok') {
					var submissionId = msg.a;
					return A2(
						$author$project$Sharecrop$Api$withSession,
						model,
						function (state) {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												activeSubmissionCommentsID: $elm$core$Maybe$Just(submissionId),
												reviewBan: false,
												reviewMessage: $elm$core$Maybe$Just('Review saved.'),
												reviewNote: '',
												reviewPartialCredit: '',
												reviewTip: '',
												reviewTipCollectibleId: '',
												submissionCommentMessage: $elm$core$Maybe$Nothing,
												submissionComments: _List_Nil
											});
									}),
								$elm$core$Platform$Cmd$batch(
									_List_fromArray(
										[
											$author$project$Sharecrop$Api$refreshAfterAccept(model),
											A2($author$project$Sharecrop$Api$fetchSubmissionComments, state.accessToken, submissionId)
										])));
						});
				} else {
					var error = msg.b.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reviewMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CollectibleNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{collectibleName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CollectibleKindChosen':
				var kind = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{collectibleKind: kind});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CollectiblePolicyChosen':
				var policy = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{collectiblePolicy: policy});
						}),
					$elm$core$Platform$Cmd$none);
			case 'MintClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$mintCommand, model, state);
					});
			case 'MintReceived':
				if (msg.a.$ === 'Ok') {
					var collectible = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										collectibleMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$View$mintSuccessLabel(collectible)),
										collectibleName: ''
									});
							}),
						$author$project$Sharecrop$Api$refreshCollectibles(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										collectibleMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CollectiblesReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									collectibles: $author$project$Sharecrop$Api$collectiblesFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AwardTaskIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{awardTaskId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AwardClicked':
				var collectibleId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$awardCommand, model, state, collectibleId);
					});
			case 'AwardReceived':
				if (msg.a.$ === 'Ok') {
					var collectible = msg.a.a;
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									awardMessage: $elm$core$Maybe$Just(
										$author$project$Sharecrop$View$awardSuccessLabel(collectible))
								});
						});
					return A2(
						$author$project$Sharecrop$Api$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								$elm$core$Platform$Cmd$batch(
									_List_fromArray(
										[
											$author$project$Sharecrop$Api$fetchCollectibles(state.accessToken),
											A5($author$project$Sharecrop$Api$fetchTasks, state.accessToken, state.taskStateFilter, state.taskListTypeFilter, state.taskListSort, state.taskListOffset)
										])));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										awardMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CollectibleCatalogReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{collectibleCatalog: response.entries});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
				}
			case 'AwardRecipientKindChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{awardRecipientId: '', awardRecipientKind: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AwardRecipientIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{awardRecipientId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AwardDefaultClicked':
				var slug = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return ($elm$core$String$trim(state.awardRecipientId) === '') ? _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											awardDefaultMessage: $elm$core$Maybe$Just('Enter a recipient id first.')
										});
								}),
							$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
							model,
							A4($author$project$Sharecrop$Api$awardDefaultCollectible, state.accessToken, slug, state.awardRecipientKind, state.awardRecipientId));
					});
			case 'AwardDefaultReceived':
				if (msg.a.$ === 'Ok') {
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									awardDefaultMessage: $elm$core$Maybe$Just('Awarded the collectible.')
								});
						});
					return _Utils_Tuple2(
						updated,
						$author$project$Sharecrop$Api$refreshCollectibles(updated));
				} else {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										awardDefaultMessage: $elm$core$Maybe$Just('Only platform admins can award default collectibles.')
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'TransferRecipientIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{transferRecipientId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TransferCollectibleClicked':
				var collectibleId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return ($elm$core$String$trim(state.transferRecipientId) === '') ? _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											transferMessage: $elm$core$Maybe$Just('Enter a recipient id first.')
										});
								}),
							$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$transferCollectible, state.accessToken, collectibleId, state.transferRecipientId));
					});
			case 'TransferCollectibleReceived':
				if (msg.a.$ === 'Ok') {
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									transferMessage: $elm$core$Maybe$Just('Transferred.')
								});
						});
					return _Utils_Tuple2(
						updated,
						$author$project$Sharecrop$Api$refreshCollectibles(updated));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										transferMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrganizationsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									organizations: $author$project$Sharecrop$Api$organizationsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createOrgName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$createOrgCommand, model, state);
					});
			case 'CreateOrgReceived':
				if (msg.a.$ === 'Ok') {
					var organization = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createOrgName: '',
										orgMessage: $elm$core$Maybe$Just('Created organization ' + organization.name)
									});
							}),
						$author$project$Sharecrop$Api$refreshOrganizations(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrgBalanceReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgBalance: $author$project$Sharecrop$Api$balanceFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'OrgLedgerReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgLedger: $author$project$Sharecrop$Api$entriesFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'PreviousOrgLedgerPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.orgLedgerOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgLedgerOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchOrganizationLedgerPage, state.accessToken, state.activeOrgId, offset));
					});
			case 'NextOrgLedgerPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.orgLedgerOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgLedgerOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchOrganizationLedgerPage, state.accessToken, state.activeOrgId, offset));
					});
			case 'OrgAuditEventsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{orgAuditEvents: response.events});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgAuditEvents: _List_Nil,
										orgTaskMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrgTeamsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgTeams: $author$project$Sharecrop$Api$teamsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'StandaloneTeamsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									standaloneTeams: $author$project$Sharecrop$Api$teamsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserDirectoryReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							if (result.$ === 'Ok') {
								var users = result.a;
								return _Utils_update(
									state,
									{userDirectory: users});
							} else {
								return _Utils_update(
									state,
									{userDirectory: _List_Nil});
							}
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserDirectoryQueryChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{userDirectoryOffset: 0, userDirectoryQuery: value});
								}),
							A3($author$project$Sharecrop$Api$fetchUserDirectoryPage, state.accessToken, value, 0));
					});
			case 'SearchUserDirectoryClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{userDirectoryOffset: 0});
								}),
							A3($author$project$Sharecrop$Api$fetchUserDirectoryPage, state.accessToken, state.userDirectoryQuery, 0));
					});
			case 'PreviousUserDirectoryPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.userDirectoryOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{userDirectoryOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchUserDirectoryPage, state.accessToken, state.userDirectoryQuery, offset));
					});
			case 'NextUserDirectoryPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.userDirectoryOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{userDirectoryOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchUserDirectoryPage, state.accessToken, state.userDirectoryQuery, offset));
					});
			case 'OrganizationQueryChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{organizationOffset: 0, organizationQuery: value});
								}),
							A3($author$project$Sharecrop$Api$fetchOrganizationsPage, state.accessToken, value, 0));
					});
			case 'SearchOrganizationsClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{organizationOffset: 0});
								}),
							A3($author$project$Sharecrop$Api$fetchOrganizationsPage, state.accessToken, state.organizationQuery, 0));
					});
			case 'PreviousOrganizationsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.organizationOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{organizationOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchOrganizationsPage, state.accessToken, state.organizationQuery, offset));
					});
			case 'NextOrganizationsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.organizationOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{organizationOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchOrganizationsPage, state.accessToken, state.organizationQuery, offset));
					});
			case 'StandaloneTeamQueryChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{standaloneTeamOffset: 0, standaloneTeamQuery: value});
								}),
							A3($author$project$Sharecrop$Api$fetchStandaloneTeamsPage, state.accessToken, value, 0));
					});
			case 'SearchStandaloneTeamsClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{standaloneTeamOffset: 0});
								}),
							A3($author$project$Sharecrop$Api$fetchStandaloneTeamsPage, state.accessToken, state.standaloneTeamQuery, 0));
					});
			case 'PreviousStandaloneTeamsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.standaloneTeamOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{standaloneTeamOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchStandaloneTeamsPage, state.accessToken, state.standaloneTeamQuery, offset));
					});
			case 'NextStandaloneTeamsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.standaloneTeamOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{standaloneTeamOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchStandaloneTeamsPage, state.accessToken, state.standaloneTeamQuery, offset));
					});
			case 'OrgTeamQueryChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTeamOffset: 0, orgTeamQuery: value});
								}),
							A4(
								$author$project$Sharecrop$Api$fetchOrgTeamsPage,
								state.accessToken,
								$author$project$Main$orgTeamSearchOrganizationID(state),
								value,
								0));
					});
			case 'SearchOrgTeamsClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTeamOffset: 0});
								}),
							A4(
								$author$project$Sharecrop$Api$fetchOrgTeamsPage,
								state.accessToken,
								$author$project$Main$orgTeamSearchOrganizationID(state),
								state.orgTeamQuery,
								0));
					});
			case 'PreviousOrgTeamsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.orgTeamOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTeamOffset: offset});
								}),
							A4(
								$author$project$Sharecrop$Api$fetchOrgTeamsPage,
								state.accessToken,
								$author$project$Main$orgTeamSearchOrganizationID(state),
								state.orgTeamQuery,
								offset));
					});
			case 'NextOrgTeamsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.orgTeamOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTeamOffset: offset});
								}),
							A4(
								$author$project$Sharecrop$Api$fetchOrgTeamsPage,
								state.accessToken,
								$author$project$Main$orgTeamSearchOrganizationID(state),
								state.orgTeamQuery,
								offset));
					});
			case 'OrgMembersReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgMembers: $author$project$Sharecrop$Api$membersFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserProfileReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							if (result.$ === 'Ok') {
								var profile = result.a;
								return _Utils_update(
									state,
									{
										userProfile: $elm$core$Maybe$Just(profile),
										userProfileError: $elm$core$Maybe$Nothing
									});
							} else {
								var error = result.a;
								return _Utils_update(
									state,
									{
										userProfile: $elm$core$Maybe$Nothing,
										userProfileError: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserWorkReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									userWork: $author$project$Sharecrop$Api$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserSubmissionsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									userSubmissions: $author$project$Sharecrop$Api$submissionsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'PreviousUserSubmissionsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v5 = state.page;
						if (_v5.$ === 'UserSubmissionsPage') {
							var userId = _v5.a;
							var offset = A2($elm$core$Basics$max, 0, state.userSubmissionsOffset - $author$project$Sharecrop$Api$selectorPageSize);
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{userSubmissionsOffset: offset});
									}),
								A3($author$project$Sharecrop$Api$fetchUserSubmissionsPage, state.accessToken, userId, offset));
						} else {
							return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
						}
					});
			case 'NextUserSubmissionsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v6 = state.page;
						if (_v6.$ === 'UserSubmissionsPage') {
							var userId = _v6.a;
							var offset = state.userSubmissionsOffset + $author$project$Sharecrop$Api$selectorPageSize;
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{userSubmissionsOffset: offset});
									}),
								A3($author$project$Sharecrop$Api$fetchUserSubmissionsPage, state.accessToken, userId, offset));
						} else {
							return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
						}
					});
			case 'StartRevisionClicked':
				var taskId = msg.a;
				var responseJson = msg.b;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									pendingRevisionResponse: responseJson,
									pendingRevisionTaskID: $elm$core$Maybe$Just(taskId)
								});
						}),
					A2($elm$browser$Browser$Navigation$pushUrl, model.key, '#/tasks/' + taskId));
			case 'SeriesListReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									seriesList: $author$project$Sharecrop$Api$seriesFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateSeriesTitleChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createSeriesTitle: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateSeriesDescriptionChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createSeriesDescription: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateSeriesClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$createSeriesCommand, model, state);
					});
			case 'SeriesDetailReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							if (result.$ === 'Ok') {
								var data = result.a;
								return _Utils_update(
									state,
									{
										seriesDetail: $elm$core$Maybe$Just(data),
										seriesDetailError: $elm$core$Maybe$Nothing,
										seriesRenameDescription: data.series.description,
										seriesRenameTitle: data.series.title
									});
							} else {
								var error = result.a;
								return _Utils_update(
									state,
									{
										seriesDetail: $elm$core$Maybe$Nothing,
										seriesDetailError: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}
						}),
					$elm$core$Platform$Cmd$none);
			case 'SeriesMutationReceived':
				if (msg.a.$ === 'Ok') {
					var data = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										addSeriesTaskId: '',
										createSeriesDescription: '',
										createSeriesTitle: '',
										seriesDetail: $elm$core$Maybe$Just(data),
										seriesMessage: $elm$core$Maybe$Just('Series saved.'),
										seriesRenameDescription: data.series.description,
										seriesRenameTitle: data.series.title
									});
							}),
						$author$project$Main$seriesListRefresh(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										seriesMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'PublishSeriesClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$seriesStateCommand, state.accessToken, seriesId, 'publish'));
					});
			case 'UnpublishSeriesClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$seriesStateCommand, state.accessToken, seriesId, 'unpublish'));
					});
			case 'CloseSeriesClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$seriesStateCommand, state.accessToken, seriesId, 'close'));
					});
			case 'ReopenSeriesClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$seriesStateCommand, state.accessToken, seriesId, 'reopen'));
					});
			case 'AddSeriesTaskIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{addSeriesTaskId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AddSeriesTaskClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$addSeriesTaskCommand, model, state, seriesId);
					});
			case 'RemoveSeriesTaskClicked':
				var seriesId = msg.a;
				var taskId = msg.b;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$removeSeriesTaskCommand, state.accessToken, seriesId, taskId));
					});
			case 'MoveSeriesTaskUpClicked':
				var seriesId = msg.a;
				var taskId = msg.b;
				return A4($author$project$Main$seriesReorder, model, seriesId, taskId, true);
			case 'MoveSeriesTaskDownClicked':
				var seriesId = msg.a;
				var taskId = msg.b;
				return A4($author$project$Main$seriesReorder, model, seriesId, taskId, false);
			case 'SeriesCommentBodyChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{seriesCommentBody: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AddSeriesCommentClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$addSeriesCommentCommand, model, state, seriesId);
					});
			case 'SeriesCommentReceived':
				if (msg.a.$ === 'Ok') {
					var comment = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										seriesCommentBody: '',
										seriesDetail: A2(
											$elm$core$Maybe$map,
											function (data) {
												return _Utils_update(
													data,
													{
														comments: _Utils_ap(
															data.comments,
															_List_fromArray(
																[comment]))
													});
											},
											state.seriesDetail)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										seriesMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SeriesRenameTitleChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{seriesRenameTitle: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SeriesRenameDescriptionChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{seriesRenameDescription: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UpdateSeriesClicked':
				var seriesId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$updateSeriesCommand, model, state, seriesId);
					});
			case 'TeamDetailReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							if (result.$ === 'Ok') {
								var detail = result.a;
								return _Utils_update(
									state,
									{
										teamDetail: $elm$core$Maybe$Just(detail),
										teamDetailError: $elm$core$Maybe$Nothing
									});
							} else {
								var error = result.a;
								return _Utils_update(
									state,
									{
										teamDetail: $elm$core$Maybe$Nothing,
										teamDetailError: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}
						}),
					$elm$core$Platform$Cmd$none);
			case 'TeamWorkReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							if (result.$ === 'Ok') {
								var response = result.a;
								return _Utils_update(
									state,
									{teamWork: response.tasks, teamWorkMessage: $elm$core$Maybe$Nothing});
							} else {
								var error = result.a;
								return _Utils_update(
									state,
									{
										teamWork: _List_Nil,
										teamWorkMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}
						}),
					$elm$core$Platform$Cmd$none);
			case 'TeamWorkQueryChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{teamWorkQuery: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TeamWorkFilterChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{teamWorkFilter: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TeamWorkTypeFilterChanged':
				var value = msg.a;
				var updated = A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{teamWorkOffset: 0, teamWorkTypeFilter: value});
					});
				return A2(
					$author$project$Sharecrop$Api$withSession,
					updated,
					function (state) {
						var _v10 = state.teamDetail;
						if (_v10.$ === 'Just') {
							var detail = _v10.a;
							return _Utils_Tuple2(
								updated,
								A6($author$project$Sharecrop$Api$fetchTeamWork, state.accessToken, detail.team.id, state.teamWorkQuery, value, state.teamWorkSort, 0));
						} else {
							return _Utils_Tuple2(updated, $elm$core$Platform$Cmd$none);
						}
					});
			case 'TeamWorkSortChanged':
				var value = msg.a;
				var updated = A2(
					$author$project$Sharecrop$Api$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{teamWorkOffset: 0, teamWorkSort: value});
					});
				return A2(
					$author$project$Sharecrop$Api$withSession,
					updated,
					function (state) {
						var _v11 = state.teamDetail;
						if (_v11.$ === 'Just') {
							var detail = _v11.a;
							return _Utils_Tuple2(
								updated,
								A6($author$project$Sharecrop$Api$fetchTeamWork, state.accessToken, detail.team.id, state.teamWorkQuery, state.teamWorkTypeFilter, value, 0));
						} else {
							return _Utils_Tuple2(updated, $elm$core$Platform$Cmd$none);
						}
					});
			case 'TeamWorkSavedViewNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{teamWorkSavedViewName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SaveTeamWorkViewClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var name = $elm$core$String$trim(state.teamWorkSavedViewName);
						if (name === '') {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												teamWorkMessage: $elm$core$Maybe$Just('A saved view name is required.')
											});
									}),
								$elm$core$Platform$Cmd$none);
						} else {
							var view = {name: name, query: state.teamWorkQuery, sort: state.teamWorkSort, stateFilter: state.teamWorkFilter, typeFilter: state.teamWorkTypeFilter};
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{teamWorkMessage: $elm$core$Maybe$Nothing});
									}),
								A3($author$project$Sharecrop$Api$saveSavedQueueView, state.accessToken, $author$project$Main$teamWorkSavedViewScope, view));
						}
					});
			case 'ApplyTeamWorkViewClicked':
				var name = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v12 = _Utils_Tuple2(
							state.teamDetail,
							A2($author$project$Main$queueViewByName, name, state.teamWorkSavedViews));
						_v12$1:
						while (true) {
							if (_v12.a.$ === 'Just') {
								if (_v12.b.$ === 'Just') {
									var detail = _v12.a.a;
									var view = _v12.b.a;
									return _Utils_Tuple2(
										A2(
											$author$project$Sharecrop$Api$updateLoggedIn,
											model,
											function (current) {
												return _Utils_update(
													current,
													{
														teamWorkFilter: view.stateFilter,
														teamWorkMessage: $elm$core$Maybe$Just('Applied view: ' + view.name),
														teamWorkOffset: 0,
														teamWorkQuery: view.query,
														teamWorkSort: view.sort,
														teamWorkTypeFilter: view.typeFilter
													});
											}),
										A6($author$project$Sharecrop$Api$fetchTeamWork, state.accessToken, detail.team.id, view.query, view.typeFilter, view.sort, 0));
								} else {
									break _v12$1;
								}
							} else {
								if (_v12.b.$ === 'Nothing') {
									break _v12$1;
								} else {
									var _v14 = _v12.a;
									return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
								}
							}
						}
						var _v13 = _v12.b;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											teamWorkMessage: $elm$core$Maybe$Just('Saved view was not found.')
										});
								}),
							$elm$core$Platform$Cmd$none);
					});
			case 'SearchTeamWorkClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v15 = state.teamDetail;
						if (_v15.$ === 'Just') {
							var detail = _v15.a;
							var offset = 0;
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{teamWorkOffset: offset});
									}),
								A6($author$project$Sharecrop$Api$fetchTeamWork, state.accessToken, detail.team.id, state.teamWorkQuery, state.teamWorkTypeFilter, state.teamWorkSort, offset));
						} else {
							return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
						}
					});
			case 'PreviousTeamWorkPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v16 = state.teamDetail;
						if (_v16.$ === 'Just') {
							var detail = _v16.a;
							var offset = A2($elm$core$Basics$max, 0, state.teamWorkOffset - $author$project$Sharecrop$Api$selectorPageSize);
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{teamWorkOffset: offset});
									}),
								A6($author$project$Sharecrop$Api$fetchTeamWork, state.accessToken, detail.team.id, state.teamWorkQuery, state.teamWorkTypeFilter, state.teamWorkSort, offset));
						} else {
							return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
						}
					});
			case 'NextTeamWorkPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v17 = state.teamDetail;
						if (_v17.$ === 'Just') {
							var detail = _v17.a;
							var offset = state.teamWorkOffset + $author$project$Sharecrop$Api$selectorPageSize;
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{teamWorkOffset: offset});
									}),
								A6($author$project$Sharecrop$Api$fetchTeamWork, state.accessToken, detail.team.id, state.teamWorkQuery, state.teamWorkTypeFilter, state.teamWorkSort, offset));
						} else {
							return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
						}
					});
			case 'TeamMemberEmailChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{teamMemberEmail: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AddTeamMemberClicked':
				var teamId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A3($author$project$Sharecrop$Api$postAddTeamMember, state.accessToken, teamId, state.teamMemberEmail));
					});
			case 'AddTeamMemberReceived':
				if (msg.a.$ === 'Ok') {
					var detail = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										teamDetail: $elm$core$Maybe$Just(detail),
										teamMemberEmail: '',
										teamMemberMessage: $elm$core$Maybe$Just('Member added.')
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										teamMemberMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrgTasksReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{orgTaskMessage: $elm$core$Maybe$Nothing, orgTasks: response.tasks});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgTaskMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										orgTasks: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrgTaskQueryChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{orgTaskQuery: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'OrgTaskFilterChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = 0;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTaskFilter: value, orgTaskOffset: offset});
								}),
							A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, state.orgTaskQuery, value, state.orgTaskTypeFilter, state.orgTaskSort, offset));
					});
			case 'OrgTaskTypeFilterChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = 0;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTaskOffset: offset, orgTaskTypeFilter: value});
								}),
							A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, state.orgTaskQuery, state.orgTaskFilter, value, state.orgTaskSort, offset));
					});
			case 'OrgTaskSortChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = 0;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTaskOffset: offset, orgTaskSort: value});
								}),
							A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, state.orgTaskQuery, state.orgTaskFilter, state.orgTaskTypeFilter, value, offset));
					});
			case 'OrgTaskSavedViewNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{orgTaskSavedViewName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SaveOrgTaskViewClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var name = $elm$core$String$trim(state.orgTaskSavedViewName);
						if (name === '') {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												orgTaskMessage: $elm$core$Maybe$Just('A saved view name is required.')
											});
									}),
								$elm$core$Platform$Cmd$none);
						} else {
							var view = {name: name, query: state.orgTaskQuery, sort: state.orgTaskSort, stateFilter: state.orgTaskFilter, typeFilter: state.orgTaskTypeFilter};
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{orgTaskMessage: $elm$core$Maybe$Nothing});
									}),
								A3($author$project$Sharecrop$Api$saveSavedQueueView, state.accessToken, $author$project$Main$orgTaskSavedViewScope, view));
						}
					});
			case 'ApplyOrgTaskViewClicked':
				var name = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var _v18 = A2($author$project$Main$queueViewByName, name, state.orgTaskSavedViews);
						if (_v18.$ === 'Just') {
							var view = _v18.a;
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												orgTaskFilter: view.stateFilter,
												orgTaskMessage: $elm$core$Maybe$Just('Applied view: ' + view.name),
												orgTaskOffset: 0,
												orgTaskQuery: view.query,
												orgTaskSort: view.sort,
												orgTaskTypeFilter: view.typeFilter
											});
									}),
								A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, view.query, view.stateFilter, view.typeFilter, view.sort, 0));
						} else {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												orgTaskMessage: $elm$core$Maybe$Just('Saved view was not found.')
											});
									}),
								$elm$core$Platform$Cmd$none);
						}
					});
			case 'SearchOrgTasksClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = 0;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTaskOffset: offset});
								}),
							A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, state.orgTaskQuery, state.orgTaskFilter, state.orgTaskTypeFilter, state.orgTaskSort, offset));
					});
			case 'PreviousOrgTasksPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.orgTaskOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTaskOffset: offset});
								}),
							A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, state.orgTaskQuery, state.orgTaskFilter, state.orgTaskTypeFilter, state.orgTaskSort, offset));
					});
			case 'NextOrgTasksPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.orgTaskOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{orgTaskOffset: offset});
								}),
							A7($author$project$Sharecrop$Api$fetchOrgTasksPage, state.accessToken, state.activeOrgId, state.orgTaskQuery, state.orgTaskFilter, state.orgTaskTypeFilter, state.orgTaskSort, offset));
					});
			case 'OrgCollectiblesReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{orgCollectibles: response.collectibles, orgCollectiblesMessage: $elm$core$Maybe$Nothing});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgCollectibles: _List_Nil,
										orgCollectiblesMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'TeamCollectiblesReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{teamCollectibles: response.collectibles, teamCollectiblesMessage: $elm$core$Maybe$Nothing});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										teamCollectibles: _List_Nil,
										teamCollectiblesMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CreateOrgTeamNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createOrgTeamName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgTeamClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$createOrgTeamCommand, model, state);
					});
			case 'CreateOrgTeamReceived':
				if (msg.a.$ === 'Ok') {
					var team = msg.a.a;
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									createOrgTeamName: '',
									orgTeamMessage: $elm$core$Maybe$Just('Created team ' + team.name)
								});
						});
					return A2(
						$author$project$Sharecrop$Api$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								A2($author$project$Sharecrop$Api$fetchOrgTeams, state.accessToken, state.activeOrgId));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgTeamMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ProvisionMemberEmailChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{provisionMemberEmail: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ToggleProvisionMemberRole':
				var role = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									provisionMemberRoles: A2($author$project$Main$toggleString, role, state.provisionMemberRoles)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ProvisionMemberClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A2($author$project$Sharecrop$Api$provisionMemberCommand, model, state);
					});
			case 'ProvisionMemberReceived':
				if (msg.a.$ === 'Ok') {
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									provisionMemberEmail: '',
									provisionMemberMessage: $elm$core$Maybe$Just('Member provisioned.')
								});
						});
					return A2(
						$author$project$Sharecrop$Api$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								A5(
									$author$project$Sharecrop$Api$authorizedRequest,
									'GET',
									state.accessToken,
									'/api/organizations/' + (state.activeOrgId + '/members'),
									$elm$http$Http$emptyBody,
									A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgMembersReceived, $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder)));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										provisionMemberMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'UpdateMemberRolesClicked':
				var userId = msg.a;
				var roles = msg.b;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A4($author$project$Sharecrop$Api$updateMemberRolesCommand, model, state, userId, roles);
					});
			case 'UpdateMemberRolesReceived':
				if (msg.a.$ === 'Ok') {
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									provisionMemberMessage: $elm$core$Maybe$Just('Member roles updated.')
								});
						});
					return A2(
						$author$project$Sharecrop$Api$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								A5(
									$author$project$Sharecrop$Api$authorizedRequest,
									'GET',
									state.accessToken,
									'/api/organizations/' + (state.activeOrgId + '/members'),
									$elm$http$Http$emptyBody,
									A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgMembersReceived, $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder)));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										provisionMemberMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'DeactivateMemberClicked':
				var userId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return A3($author$project$Sharecrop$Api$deactivateMemberCommand, model, state, userId);
					});
			case 'DeactivateMemberReceived':
				if (msg.a.$ === 'Ok') {
					var updated = A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									provisionMemberMessage: $elm$core$Maybe$Just('Member deactivated.')
								});
						});
					return A2(
						$author$project$Sharecrop$Api$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								A5(
									$author$project$Sharecrop$Api$authorizedRequest,
									'GET',
									state.accessToken,
									'/api/organizations/' + (state.activeOrgId + '/members'),
									$elm$http$Http$emptyBody,
									A2($elm$http$Http$expectJson, $author$project$Sharecrop$Types$OrgMembersReceived, $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder)));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										provisionMemberMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CreateTaskOwnerChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createTaskOwner: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateTaskTypeChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							var _v19 = $author$project$Sharecrop$View$taskTemplate(value);
							if (_v19.$ === 'Just') {
								var template = _v19.a;
								return _Utils_update(
									state,
									{createDescription: template.description, createResponseSchema: template.schema, createSchemaFields: _List_Nil, createTaskType: value});
							} else {
								return _Utils_update(
									state,
									{createResponseSchema: '{\"kind\":\"freeform\"}', createSchemaFields: _List_Nil, createTaskType: value});
							}
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateReferenceURLChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createReferenceURL: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TaskCommentBodyChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{taskCommentBody: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AddTaskCommentClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return ($elm$core$String$trim(state.taskCommentBody) === '') ? _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											taskCommentMessage: $elm$core$Maybe$Just('Write a comment first.')
										});
								}),
							$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{taskCommentMessage: $elm$core$Maybe$Nothing});
								}),
							A3(
								$author$project$Sharecrop$Api$postTaskComment,
								state.accessToken,
								taskId,
								$elm$core$String$trim(state.taskCommentBody)));
					});
			case 'TaskCommentReceived':
				if (msg.a.$ === 'Ok') {
					var comment = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskCommentBody: '',
										taskCommentMessage: $elm$core$Maybe$Nothing,
										taskComments: _Utils_ap(
											state.taskComments,
											_List_fromArray(
												[comment]))
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										taskCommentMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'TaskCommentsReceived':
				if (msg.a.$ === 'Ok') {
					var comments = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{taskComments: comments});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{taskComments: _List_Nil});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OpenSubmissionComments':
				var submissionId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											activeSubmissionCommentsID: $elm$core$Maybe$Just(submissionId),
											submissionCommentMessage: $elm$core$Maybe$Nothing,
											submissionComments: _List_Nil
										});
								}),
							A2($author$project$Sharecrop$Api$fetchSubmissionComments, state.accessToken, submissionId));
					});
			case 'SubmissionCommentsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{submissionComments: response.comments});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submissionCommentMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SubmissionCommentBodyChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{submissionCommentBody: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AddSubmissionCommentClicked':
				var submissionId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return ($elm$core$String$trim(state.submissionCommentBody) === '') ? _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{
											submissionCommentMessage: $elm$core$Maybe$Just('Write a comment first.')
										});
								}),
							$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{submissionCommentMessage: $elm$core$Maybe$Nothing});
								}),
							A3(
								$author$project$Sharecrop$Api$addSubmissionComment,
								state.accessToken,
								submissionId,
								$elm$core$String$trim(state.submissionCommentBody)));
					});
			case 'SubmissionCommentAdded':
				if (msg.a.$ === 'Ok') {
					return A2(
						$author$project$Sharecrop$Api$withSession,
						model,
						function (state) {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{submissionCommentBody: ''});
									}),
								function () {
									var _v20 = state.activeSubmissionCommentsID;
									if (_v20.$ === 'Just') {
										var submissionId = _v20.a;
										return A2($author$project$Sharecrop$Api$fetchSubmissionComments, state.accessToken, submissionId);
									} else {
										return $elm$core$Platform$Cmd$none;
									}
								}());
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submissionCommentMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AccountEmailChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{accountEmail: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CurrentPasswordChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{currentPassword: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'NewPasswordChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{newPassword: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'EmailVerificationInputChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{emailVerificationInput: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'RequestEmailVerificationClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{accountMessage: $elm$core$Maybe$Nothing});
								}),
							$author$project$Sharecrop$Api$requestEmailVerification(state.accessToken));
					});
			case 'ConfirmEmailVerificationClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{accountMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Sharecrop$Api$confirmEmailVerification, state.accessToken, state.emailVerificationInput));
					});
			case 'UpdateProfileClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{accountMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Sharecrop$Api$updateProfile, state.accessToken, state.accountEmail));
					});
			case 'ChangePasswordClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{accountMessage: $elm$core$Maybe$Nothing});
								}),
							A3($author$project$Sharecrop$Api$changePassword, state.accessToken, state.currentPassword, state.newPassword));
					});
			case 'DeactivateAccountClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{accountMessage: $elm$core$Maybe$Nothing});
								}),
							$author$project$Sharecrop$Api$deactivateAccount(state.accessToken));
					});
			case 'PrivacyRequestClicked':
				var kind = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{accountMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Sharecrop$Api$requestPrivacy, state.accessToken, kind));
					});
			case 'EmailVerificationRequested':
				if (msg.a.$ === 'Ok') {
					var token = msg.a.a;
					return (token === '') ? _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just('Verification instructions sent.')
									});
							}),
						$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just('Verification token created.'),
										emailVerificationInput: token,
										emailVerificationToken: token
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AccountActionReceived':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just('Account updated.'),
										currentPassword: '',
										emailVerificationInput: '',
										newPassword: ''
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'DeactivateAccountReceived':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{email: '', password: '', session: $author$project$Sharecrop$Types$LoggedOut}),
						A2($elm$browser$Browser$Navigation$pushUrl, model.key, '#/'));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'PrivacyRequestReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just('Privacy request queued: ' + response.kind)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										accountMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SavedQueueViewsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					var teamViews = A2(
						$elm$core$List$map,
						$author$project$Main$queueViewFromResponse,
						A2(
							$elm$core$List$filter,
							function (view) {
								return _Utils_eq(view.scope, $author$project$Main$teamWorkSavedViewScope);
							},
							response.views));
					var orgViews = A2(
						$elm$core$List$map,
						$author$project$Main$queueViewFromResponse,
						A2(
							$elm$core$List$filter,
							function (view) {
								return _Utils_eq(view.scope, $author$project$Main$orgTaskSavedViewScope);
							},
							response.views));
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{orgTaskSavedViews: orgViews, teamWorkSavedViews: teamViews});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
				}
			case 'SavedQueueViewSaved':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					var view = $author$project$Main$queueViewFromResponse(response);
					return _Utils_eq(response.scope, $author$project$Main$teamWorkSavedViewScope) ? _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										teamWorkMessage: $elm$core$Maybe$Just('Saved view: ' + view.name),
										teamWorkSavedViewName: '',
										teamWorkSavedViews: A2($author$project$Main$saveQueueView, view, state.teamWorkSavedViews)
									});
							}),
						$elm$core$Platform$Cmd$none) : (_Utils_eq(response.scope, $author$project$Main$orgTaskSavedViewScope) ? _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgTaskMessage: $elm$core$Maybe$Just('Saved view: ' + view.name),
										orgTaskSavedViewName: '',
										orgTaskSavedViews: A2($author$project$Main$saveQueueView, view, state.orgTaskSavedViews)
									});
							}),
						$elm$core$Platform$Cmd$none) : _Utils_Tuple2(model, $elm$core$Platform$Cmd$none));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgTaskMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										teamWorkMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OperationsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Nothing,
										operations: $elm$core$Maybe$Just(response)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										operations: $elm$core$Maybe$Nothing
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AuditEventsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{adminMessage: $elm$core$Maybe$Nothing, auditEvents: response.events});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										auditEvents: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'PlatformAdminsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{platformAdmins: response.admins});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										platformAdmins: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AdminSelectedUserChanged':
				var userId = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{adminSelectedUserId: userId});
						}),
					$elm$core$Platform$Cmd$none);
			case 'GrantPlatformAdminClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Sharecrop$Api$grantPlatformAdmin, state.accessToken, state.adminSelectedUserId));
					});
			case 'PlatformAdminGranted':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return A2(
						$author$project$Sharecrop$Api$withSession,
						model,
						function (state) {
							return _Utils_Tuple2(
								A2(
									$author$project$Sharecrop$Api$updateLoggedIn,
									model,
									function (current) {
										return _Utils_update(
											current,
											{
												adminMessage: $elm$core$Maybe$Just('Platform admin granted.'),
												adminSelectedUserId: '',
												platformAdminsOffset: 0
											});
									}),
								A2($author$project$Sharecrop$Api$fetchPlatformAdmins, state.accessToken, 0));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'RevokePlatformAdminClicked':
				var userID = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Sharecrop$Api$revokePlatformAdmin, state.accessToken, userID));
					});
			case 'PlatformAdminRevoked':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just('Platform admin revoked.'),
										platformAdmins: A2($author$project$Main$removePlatformAdmin, response.userID, state.platformAdmins)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AdminModerationReportsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{adminMessage: $elm$core$Maybe$Nothing, adminModerationReports: response.reports});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										adminModerationReports: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AdminModerationStateFilterChanged':
				var value = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminModerationOffset: 0, adminModerationStateFilter: value});
								}),
							A3($author$project$Sharecrop$Api$fetchAdminModerationReports, state.accessToken, value, 0));
					});
			case 'PreviousAdminModerationPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.adminModerationOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminModerationOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchAdminModerationReports, state.accessToken, state.adminModerationStateFilter, offset));
					});
			case 'NextAdminModerationPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.adminModerationOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminModerationOffset: offset});
								}),
							A3($author$project$Sharecrop$Api$fetchAdminModerationReports, state.accessToken, state.adminModerationStateFilter, offset));
					});
			case 'AdminModerationResolutionNoteChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{adminModerationResolutionNote: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TriageModerationReportClicked':
				var reportID = msg.a;
				var stateValue = msg.b;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminMessage: $elm$core$Maybe$Nothing});
								}),
							A4($author$project$Sharecrop$Api$triageModerationReport, state.accessToken, reportID, stateValue, state.adminModerationResolutionNote));
					});
			case 'AdminModerationReportTriaged':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just('Moderation report updated.'),
										adminModerationReports: A2($author$project$Main$replaceModerationReport, response, state.adminModerationReports),
										adminModerationResolutionNote: ''
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AdminPrivacyRequestsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{adminMessage: $elm$core$Maybe$Nothing, adminPrivacyRequests: response.requests});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										adminPrivacyRequests: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'PreviousAdminPrivacyPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.adminPrivacyOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminPrivacyOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchAdminPrivacyRequests, state.accessToken, offset));
					});
			case 'NextAdminPrivacyPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.adminPrivacyOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminPrivacyOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchAdminPrivacyRequests, state.accessToken, offset));
					});
			case 'AdminPrivacyResolutionNoteChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{adminPrivacyResolutionNote: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'RunPrivacyRetentionClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminMessage: $elm$core$Maybe$Nothing});
								}),
							$author$project$Sharecrop$Api$runPrivacyRetention(state.accessToken));
					});
			case 'PrivacyRetentionRunReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just('Privacy retention run finished.'),
										adminRetentionRedactedFieldCount: $elm$core$Maybe$Just(response.redactedFieldCount)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ResolveAdminPrivacyRequestClicked':
				var requestId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{adminMessage: $elm$core$Maybe$Nothing});
								}),
							A3($author$project$Sharecrop$Api$resolveAdminPrivacyRequest, state.accessToken, requestId, state.adminPrivacyResolutionNote));
					});
			case 'AdminPrivacyRequestResolved':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just('Privacy request resolved.'),
										adminPrivacyRequests: A2($author$project$Main$replacePrivacyRequest, response, state.adminPrivacyRequests),
										adminPrivacyResolutionNote: ''
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										adminMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AuditActionFilterChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{auditActionFilter: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AuditSubjectKindFilterChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{auditSubjectKindFilter: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AuditSubjectIDFilterChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Sharecrop$Api$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{auditSubjectIDFilter: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SearchAuditEventsClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{auditEventsOffset: 0});
								}),
							A5($author$project$Sharecrop$Api$fetchAuditEvents, state.accessToken, state.auditActionFilter, state.auditSubjectKindFilter, state.auditSubjectIDFilter, 0));
					});
			case 'PreviousAuditEventsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.auditEventsOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{auditEventsOffset: offset});
								}),
							A5($author$project$Sharecrop$Api$fetchAuditEvents, state.accessToken, state.auditActionFilter, state.auditSubjectKindFilter, state.auditSubjectIDFilter, offset));
					});
			case 'NextAuditEventsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.auditEventsOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{auditEventsOffset: offset});
								}),
							A5($author$project$Sharecrop$Api$fetchAuditEvents, state.accessToken, state.auditActionFilter, state.auditSubjectKindFilter, state.auditSubjectIDFilter, offset));
					});
			case 'PreviousPlatformAdminsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.platformAdminsOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{platformAdminsOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchPlatformAdmins, state.accessToken, offset));
					});
			case 'NextPlatformAdminsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.platformAdminsOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{platformAdminsOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchPlatformAdmins, state.accessToken, offset));
					});
			case 'NotificationsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{inboxMessage: $elm$core$Maybe$Nothing, notifications: response.notifications});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										inboxMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error)),
										notifications: _List_Nil
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'PreviousNotificationsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = A2($elm$core$Basics$max, 0, state.notificationsOffset - $author$project$Sharecrop$Api$selectorPageSize);
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{notificationsOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchNotifications, state.accessToken, offset));
					});
			case 'NextNotificationsPageClicked':
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						var offset = state.notificationsOffset + $author$project$Sharecrop$Api$selectorPageSize;
						return _Utils_Tuple2(
							A2(
								$author$project$Sharecrop$Api$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{notificationsOffset: offset});
								}),
							A2($author$project$Sharecrop$Api$fetchNotifications, state.accessToken, offset));
					});
			case 'MarkNotificationReadClicked':
				var notificationId = msg.a;
				return A2(
					$author$project$Sharecrop$Api$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Sharecrop$Api$markNotificationRead, state.accessToken, notificationId));
					});
			case 'NotificationReadReceived':
				if (msg.a.$ === 'Ok') {
					var notification = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										inboxMessage: $elm$core$Maybe$Nothing,
										notifications: A2($author$project$Main$replaceNotification, notification, state.notifications)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Sharecrop$Api$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										inboxMessage: $elm$core$Maybe$Just(
											$author$project$Sharecrop$Labels$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'LinkClicked':
				var request = msg.a;
				if (request.$ === 'Internal') {
					var url = request.a;
					return _Utils_Tuple2(
						model,
						A2(
							$elm$browser$Browser$Navigation$pushUrl,
							model.key,
							$elm$url$Url$toString(url)));
				} else {
					var href = request.a;
					return _Utils_Tuple2(
						model,
						$elm$browser$Browser$Navigation$load(href));
				}
			case 'UrlChanged':
				var url = msg.a;
				var page = $author$project$Main$pageFromUrl(url);
				var _v22 = model.session;
				if (_v22.$ === 'LoggedIn') {
					var state = _v22.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								route: page,
								session: $author$project$Sharecrop$Types$LoggedIn(
									A2($author$project$Main$enterPage, page, state))
							}),
						A3($author$project$Sharecrop$Api$routeLoadCmd, state.accessToken, state.subjectId, page));
				} else {
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{route: page}),
						$elm$core$Platform$Cmd$none);
				}
			default:
				return _Utils_Tuple2(
					model,
					$author$project$Main$reloadDemo(_Utils_Tuple0));
		}
	});
var $elm$html$Html$Attributes$stringProperty = F2(
	function (key, string) {
		return A2(
			_VirtualDom_property,
			key,
			$elm$json$Json$Encode$string(string));
	});
var $elm$html$Html$Attributes$class = $elm$html$Html$Attributes$stringProperty('className');
var $elm$html$Html$div = _VirtualDom_node('div');
var $elm$html$Html$main_ = _VirtualDom_node('main');
var $elm$html$Html$h1 = _VirtualDom_node('h1');
var $elm$virtual_dom$VirtualDom$text = _VirtualDom_text;
var $elm$html$Html$text = $elm$virtual_dom$VirtualDom$text;
var $author$project$Sharecrop$Ui$pageTitle = function (title) {
	return A2(
		$elm$html$Html$h1,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-3xl font-semibold')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(title)
			]));
};
var $author$project$Sharecrop$Types$ConfirmPasswordResetClicked = {$: 'ConfirmPasswordResetClicked'};
var $author$project$Sharecrop$Types$EmailChanged = function (a) {
	return {$: 'EmailChanged', a: a};
};
var $author$project$Sharecrop$Types$GuestClicked = {$: 'GuestClicked'};
var $author$project$Sharecrop$Types$LoginClicked = {$: 'LoginClicked'};
var $author$project$Sharecrop$Types$PasswordChanged = function (a) {
	return {$: 'PasswordChanged', a: a};
};
var $author$project$Sharecrop$Types$PasswordResetEmailChanged = function (a) {
	return {$: 'PasswordResetEmailChanged', a: a};
};
var $author$project$Sharecrop$Types$PasswordResetPasswordChanged = function (a) {
	return {$: 'PasswordResetPasswordChanged', a: a};
};
var $author$project$Sharecrop$Types$PasswordResetTokenChanged = function (a) {
	return {$: 'PasswordResetTokenChanged', a: a};
};
var $author$project$Sharecrop$Types$RegisterClicked = {$: 'RegisterClicked'};
var $author$project$Sharecrop$Types$RequestPasswordResetClicked = {$: 'RequestPasswordResetClicked'};
var $elm$html$Html$form = _VirtualDom_node('form');
var $elm$html$Html$p = _VirtualDom_node('p');
var $author$project$Sharecrop$Ui$label_ = function (value) {
	return A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm uppercase tracking-wide text-slate-600')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(value)
			]));
};
var $elm$virtual_dom$VirtualDom$attribute = F2(
	function (key, value) {
		return A2(
			_VirtualDom_attribute,
			_VirtualDom_noOnOrFormAction(key),
			_VirtualDom_noJavaScriptOrHtmlUri(value));
	});
var $elm$html$Html$Attributes$attribute = $elm$virtual_dom$VirtualDom$attribute;
var $author$project$Sharecrop$Ui$testId = function (value) {
	return A2($elm$html$Html$Attributes$attribute, 'data-testid', value);
};
var $author$project$Sharecrop$Ui$errorText = F2(
	function (identifier, message) {
		return A2(
			$elm$html$Html$p,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-sm text-red-600'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(message)
				]));
	});
var $author$project$Sharecrop$View$maybeError = F2(
	function (message, identifier) {
		if (message.$ === 'Just') {
			var value = message.a;
			return A2($author$project$Sharecrop$Ui$errorText, identifier, value);
		} else {
			return $elm$html$Html$text('');
		}
	});
var $elm$virtual_dom$VirtualDom$Normal = function (a) {
	return {$: 'Normal', a: a};
};
var $elm$virtual_dom$VirtualDom$on = _VirtualDom_on;
var $elm$html$Html$Events$on = F2(
	function (event, decoder) {
		return A2(
			$elm$virtual_dom$VirtualDom$on,
			event,
			$elm$virtual_dom$VirtualDom$Normal(decoder));
	});
var $elm$html$Html$Events$onClick = function (msg) {
	return A2(
		$elm$html$Html$Events$on,
		'click',
		$elm$json$Json$Decode$succeed(msg));
};
var $elm$html$Html$Events$alwaysStop = function (x) {
	return _Utils_Tuple2(x, true);
};
var $elm$virtual_dom$VirtualDom$MayStopPropagation = function (a) {
	return {$: 'MayStopPropagation', a: a};
};
var $elm$html$Html$Events$stopPropagationOn = F2(
	function (event, decoder) {
		return A2(
			$elm$virtual_dom$VirtualDom$on,
			event,
			$elm$virtual_dom$VirtualDom$MayStopPropagation(decoder));
	});
var $elm$json$Json$Decode$at = F2(
	function (fields, decoder) {
		return A3($elm$core$List$foldr, $elm$json$Json$Decode$field, decoder, fields);
	});
var $elm$html$Html$Events$targetValue = A2(
	$elm$json$Json$Decode$at,
	_List_fromArray(
		['target', 'value']),
	$elm$json$Json$Decode$string);
var $elm$html$Html$Events$onInput = function (tagger) {
	return A2(
		$elm$html$Html$Events$stopPropagationOn,
		'input',
		A2(
			$elm$json$Json$Decode$map,
			$elm$html$Html$Events$alwaysStop,
			A2($elm$json$Json$Decode$map, tagger, $elm$html$Html$Events$targetValue)));
};
var $elm$html$Html$Events$alwaysPreventDefault = function (msg) {
	return _Utils_Tuple2(msg, true);
};
var $elm$virtual_dom$VirtualDom$MayPreventDefault = function (a) {
	return {$: 'MayPreventDefault', a: a};
};
var $elm$html$Html$Events$preventDefaultOn = F2(
	function (event, decoder) {
		return A2(
			$elm$virtual_dom$VirtualDom$on,
			event,
			$elm$virtual_dom$VirtualDom$MayPreventDefault(decoder));
	});
var $elm$html$Html$Events$onSubmit = function (msg) {
	return A2(
		$elm$html$Html$Events$preventDefaultOn,
		'submit',
		A2(
			$elm$json$Json$Decode$map,
			$elm$html$Html$Events$alwaysPreventDefault,
			$elm$json$Json$Decode$succeed(msg)));
};
var $elm$html$Html$Attributes$placeholder = $elm$html$Html$Attributes$stringProperty('placeholder');
var $elm$html$Html$button = _VirtualDom_node('button');
var $author$project$Sharecrop$Ui$primaryButtonClass = 'inline-flex min-h-[44px] items-center justify-center rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50';
var $author$project$Sharecrop$Ui$primaryButton = F2(
	function (attrs, labelText) {
		return A2(
			$elm$html$Html$button,
			A2(
				$elm$core$List$cons,
				$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$primaryButtonClass),
				attrs),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Sharecrop$Ui$secondaryButtonClass = 'inline-flex min-h-[44px] items-center justify-center rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100';
var $author$project$Sharecrop$Ui$secondaryButton = F2(
	function (attrs, labelText) {
		return A2(
			$elm$html$Html$button,
			A2(
				$elm$core$List$cons,
				$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
				attrs),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Sharecrop$Ui$fieldClass = 'w-full min-h-[44px] rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none';
var $elm$html$Html$input = _VirtualDom_node('input');
var $author$project$Sharecrop$Ui$textInput = function (attrs) {
	return A2(
		$elm$html$Html$input,
		A2(
			$elm$core$List$cons,
			$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
			attrs),
		_List_Nil);
};
var $elm$html$Html$Attributes$type_ = $elm$html$Html$Attributes$stringProperty('type');
var $elm$html$Html$Attributes$value = $elm$html$Html$Attributes$stringProperty('value');
var $author$project$Sharecrop$View$authView = function (model) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$LoginClicked)
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-slate-600')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Sign in or create an account to view your credit ledger and set up agents.')
					])),
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('email'),
						$elm$html$Html$Attributes$placeholder('Email'),
						$elm$html$Html$Attributes$value(model.email),
						$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$EmailChanged),
						$author$project$Sharecrop$Ui$testId('email')
					])),
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('password'),
						$elm$html$Html$Attributes$placeholder('Password'),
						$elm$html$Html$Attributes$value(model.password),
						$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$PasswordChanged),
						$author$project$Sharecrop$Ui$testId('password')
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex gap-3')
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('login')
							]),
						'Log in'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$RegisterClicked),
								$author$project$Sharecrop$Ui$testId('register')
							]),
						'Register'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$GuestClicked),
								$author$project$Sharecrop$Ui$testId('guest-login')
							]),
						'Continue as guest')
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2 border-t border-slate-100 pt-4')
					]),
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$label_('Password reset'),
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('email'),
								$elm$html$Html$Attributes$placeholder('Account email'),
								$elm$html$Html$Attributes$value(model.resetEmail),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$PasswordResetEmailChanged),
								$author$project$Sharecrop$Ui$testId('reset-email')
							])),
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('button'),
										$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$RequestPasswordResetClicked),
										$author$project$Sharecrop$Ui$testId('request-password-reset')
									]),
								'Create reset token')
							])),
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('text'),
								$elm$html$Html$Attributes$placeholder('Reset token'),
								$elm$html$Html$Attributes$value(model.resetToken),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$PasswordResetTokenChanged),
								$author$project$Sharecrop$Ui$testId('reset-token')
							])),
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('password'),
								$elm$html$Html$Attributes$placeholder('New password'),
								$elm$html$Html$Attributes$value(model.resetPassword),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$PasswordResetPasswordChanged),
								$author$project$Sharecrop$Ui$testId('reset-password')
							])),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$ConfirmPasswordResetClicked),
								$author$project$Sharecrop$Ui$testId('confirm-password-reset')
							]),
						'Reset password')
					])),
				A2($author$project$Sharecrop$View$maybeError, model.authError, 'auth-error')
			]));
};
var $author$project$Sharecrop$Types$LogoutClicked = {$: 'LogoutClicked'};
var $author$project$Sharecrop$Types$ResetDemoClicked = {$: 'ResetDemoClicked'};
var $elm$html$Html$a = _VirtualDom_node('a');
var $elm$html$Html$Attributes$href = function (url) {
	return A2(
		$elm$html$Html$Attributes$stringProperty,
		'href',
		_VirtualDom_noJavaScriptUri(url));
};
var $author$project$Sharecrop$Types$pageToPath = function (page) {
	switch (page.$) {
		case 'OverviewPage':
			return '/';
		case 'TasksPage':
			return '/tasks';
		case 'CreateTaskPage':
			return '/tasks/new';
		case 'TaskDetailPage':
			var taskId = page.a;
			return '/tasks/' + taskId;
		case 'DiscoveryPage':
			return '/discovery';
		case 'FundingPage':
			return '/funding';
		case 'AgentsPage':
			return '/agents';
		case 'CollectiblesPage':
			return '/collectibles';
		case 'OrganizationsPage':
			return '/organizations';
		case 'OrganizationDetailPage':
			var organizationId = page.a;
			return '/organizations/' + organizationId;
		case 'UserDetailPage':
			var userId = page.a;
			return '/users/' + userId;
		case 'UserWorkPage':
			var userId = page.a;
			return '/users/' + (userId + '/work');
		case 'UserSubmissionsPage':
			var userId = page.a;
			return '/users/' + (userId + '/submissions');
		case 'CollectibleDetailPage':
			var collectibleId = page.a;
			return '/collectibles/' + collectibleId;
		case 'SeriesListPage':
			return '/series';
		case 'SeriesDetailPage':
			var seriesId = page.a;
			return '/series/' + seriesId;
		case 'TeamDetailPage':
			var teamId = page.a;
			return '/teams/' + teamId;
		case 'AdminPage':
			return '/admin';
		case 'InboxPage':
			return '/inbox';
		default:
			return '/not-found';
	}
};
var $author$project$Sharecrop$View$navLink = F4(
	function (current, target, identifier, labelText) {
		var styleClass = _Utils_eq(
			$author$project$Sharecrop$Types$pageToPath(current),
			$author$project$Sharecrop$Types$pageToPath(target)) ? $author$project$Sharecrop$Ui$primaryButtonClass : $author$project$Sharecrop$Ui$secondaryButtonClass;
		return A2(
			$elm$html$Html$a,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$href(
					'#' + $author$project$Sharecrop$Types$pageToPath(target)),
					$elm$html$Html$Attributes$class(styleClass),
					$author$project$Sharecrop$Ui$testId('nav-' + identifier)
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Sharecrop$View$navBar = F4(
	function (demo, current, subjectId, isAdmin) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
				]),
			_List_fromArray(
				[
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$OverviewPage, 'overview', 'Overview'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$TasksPage, 'tasks', 'Tasks'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$CreateTaskPage, 'create-task', 'New task'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$DiscoveryPage, 'discovery', 'Discovery'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$FundingPage, 'funding', 'Funding'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$AgentsPage, 'agents', 'Agents'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$CollectiblesPage, 'collectibles', 'Collectibles'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$InboxPage, 'inbox', 'Inbox'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$SeriesListPage, 'series-list', 'Series'),
					A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$OrganizationsPage, 'organizations', 'Organizations'),
					isAdmin ? A4($author$project$Sharecrop$View$navLink, current, $author$project$Sharecrop$Types$AdminPage, 'admin', 'Admin') : $elm$html$Html$text(''),
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/users/' + subjectId),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
							$author$project$Sharecrop$Ui$testId('nav-profile')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Profile')
						])),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$LogoutClicked),
							$author$project$Sharecrop$Ui$testId('logout')
						]),
					'Log out'),
					demo ? A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$ResetDemoClicked),
							$author$project$Sharecrop$Ui$testId('reset-demo')
						]),
					'Reset demo') : $elm$html$Html$text('')
				]));
	});
var $elm$virtual_dom$VirtualDom$keyedNode = function (tag) {
	return _VirtualDom_keyedNode(
		_VirtualDom_noScript(tag));
};
var $elm$html$Html$Keyed$node = $elm$virtual_dom$VirtualDom$keyedNode;
var $author$project$Sharecrop$Types$AdminModerationResolutionNoteChanged = function (a) {
	return {$: 'AdminModerationResolutionNoteChanged', a: a};
};
var $author$project$Sharecrop$Types$AdminModerationStateFilterChanged = function (a) {
	return {$: 'AdminModerationStateFilterChanged', a: a};
};
var $author$project$Sharecrop$Types$AdminPrivacyResolutionNoteChanged = function (a) {
	return {$: 'AdminPrivacyResolutionNoteChanged', a: a};
};
var $author$project$Sharecrop$Types$AdminSelectedUserChanged = function (a) {
	return {$: 'AdminSelectedUserChanged', a: a};
};
var $author$project$Sharecrop$Types$AuditActionFilterChanged = function (a) {
	return {$: 'AuditActionFilterChanged', a: a};
};
var $author$project$Sharecrop$Types$AuditSubjectIDFilterChanged = function (a) {
	return {$: 'AuditSubjectIDFilterChanged', a: a};
};
var $author$project$Sharecrop$Types$AuditSubjectKindFilterChanged = function (a) {
	return {$: 'AuditSubjectKindFilterChanged', a: a};
};
var $author$project$Sharecrop$Types$GrantPlatformAdminClicked = {$: 'GrantPlatformAdminClicked'};
var $author$project$Sharecrop$Types$NextAdminModerationPageClicked = {$: 'NextAdminModerationPageClicked'};
var $author$project$Sharecrop$Types$NextAdminPrivacyPageClicked = {$: 'NextAdminPrivacyPageClicked'};
var $author$project$Sharecrop$Types$NextAuditEventsPageClicked = {$: 'NextAuditEventsPageClicked'};
var $author$project$Sharecrop$Types$NextPlatformAdminsPageClicked = {$: 'NextPlatformAdminsPageClicked'};
var $author$project$Sharecrop$Types$PreviousAdminModerationPageClicked = {$: 'PreviousAdminModerationPageClicked'};
var $author$project$Sharecrop$Types$PreviousAdminPrivacyPageClicked = {$: 'PreviousAdminPrivacyPageClicked'};
var $author$project$Sharecrop$Types$PreviousAuditEventsPageClicked = {$: 'PreviousAuditEventsPageClicked'};
var $author$project$Sharecrop$Types$PreviousPlatformAdminsPageClicked = {$: 'PreviousPlatformAdminsPageClicked'};
var $author$project$Sharecrop$Types$RunPrivacyRetentionClicked = {$: 'RunPrivacyRetentionClicked'};
var $author$project$Sharecrop$Types$SearchAuditEventsClicked = {$: 'SearchAuditEventsClicked'};
var $author$project$Sharecrop$Types$TriageModerationReportClicked = F2(
	function (a, b) {
		return {$: 'TriageModerationReportClicked', a: a, b: b};
	});
var $elm$html$Html$span = _VirtualDom_node('span');
var $author$project$Sharecrop$Ui$badge = function (value) {
	return A2(
		$elm$html$Html$span,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('inline-flex items-center rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-700')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(value)
			]));
};
var $elm$html$Html$Attributes$boolProperty = F2(
	function (key, bool) {
		return A2(
			_VirtualDom_property,
			key,
			$elm$json$Json$Encode$bool(bool));
	});
var $elm$html$Html$Attributes$disabled = $elm$html$Html$Attributes$boolProperty('disabled');
var $elm$html$Html$dl = _VirtualDom_node('dl');
var $author$project$Sharecrop$View$emptyLabel = function (value) {
	return ($elm$core$String$trim(value) === '') ? 'none' : value;
};
var $elm$html$Html$dd = _VirtualDom_node('dd');
var $elm$html$Html$dt = _VirtualDom_node('dt');
var $author$project$Sharecrop$View$operationFact = F2(
	function (labelText, valueText) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('rounded border border-slate-200 p-2')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$dt,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs font-semibold text-slate-500')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(labelText)
						])),
					A2(
					$elm$html$Html$dd,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('break-words text-slate-900')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(valueText)
						]))
				]));
	});
var $author$project$Sharecrop$View$adminModerationReportRow = F2(
	function (resolutionNote, report) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2 py-3 text-sm'),
					$author$project$Sharecrop$Ui$testId('admin-moderation-report')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$badge(report.state),
							$author$project$Sharecrop$Ui$badge(report.reason),
							A2(
							$elm$html$Html$span,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('font-medium text-slate-900')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(report.subjectKind)
								])),
							(report.subjectHref === '') ? A2(
							$elm$html$Html$span,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('break-all text-xs text-slate-500')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(report.subjectID)
								])) : A2(
							$elm$html$Html$a,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$href(report.subjectHref),
									$elm$html$Html$Attributes$class('break-all text-xs font-medium text-emerald-700'),
									$author$project$Sharecrop$Ui$testId('admin-moderation-subject-link')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(report.subjectID)
								]))
						])),
					A2(
					$elm$html$Html$dl,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('grid gap-2 sm:grid-cols-2')
						]),
					_List_fromArray(
						[
							A2($author$project$Sharecrop$View$operationFact, 'Reporter', report.reporterUserID),
							A2($author$project$Sharecrop$View$operationFact, 'Created', report.createdAt),
							A2(
							$author$project$Sharecrop$View$operationFact,
							'Updated by',
							$author$project$Sharecrop$View$emptyLabel(report.updatedBy)),
							A2(
							$author$project$Sharecrop$View$operationFact,
							'Updated',
							$author$project$Sharecrop$View$emptyLabel(report.updatedAt))
						])),
					(report.resolutionNote === '') ? $elm$html$Html$text('') : A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs text-slate-600'),
							$author$project$Sharecrop$Ui$testId('admin-moderation-resolution-note')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(report.resolutionNote)
						])),
					(report.details === '') ? $elm$html$Html$text('') : A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-700 break-words'),
							$author$project$Sharecrop$Ui$testId('admin-moderation-details')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(report.details)
						])),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									A2($author$project$Sharecrop$Types$TriageModerationReportClicked, report.id, 'open')),
									$author$project$Sharecrop$Ui$testId('admin-moderation-open')
								]),
							'Reopen'),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									A2($author$project$Sharecrop$Types$TriageModerationReportClicked, report.id, 'resolved')),
									$elm$html$Html$Attributes$disabled(
									$elm$core$String$trim(resolutionNote) === ''),
									$author$project$Sharecrop$Ui$testId('admin-moderation-resolve')
								]),
							'Resolve'),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									A2($author$project$Sharecrop$Types$TriageModerationReportClicked, report.id, 'dismissed')),
									$elm$html$Html$Attributes$disabled(
									$elm$core$String$trim(resolutionNote) === ''),
									$author$project$Sharecrop$Ui$testId('admin-moderation-dismiss')
								]),
							'Dismiss')
						]))
				]));
	});
var $author$project$Sharecrop$Types$ResolveAdminPrivacyRequestClicked = function (a) {
	return {$: 'ResolveAdminPrivacyRequestClicked', a: a};
};
var $author$project$Sharecrop$Ui$codeBlockClass = 'whitespace-pre-wrap break-words rounded-md bg-slate-900 p-3 text-xs text-slate-100';
var $elm$html$Html$pre = _VirtualDom_node('pre');
var $author$project$Sharecrop$Ui$codeBlock = F2(
	function (attrs, value) {
		return A2(
			$elm$html$Html$pre,
			A2(
				$elm$core$List$cons,
				$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$codeBlockClass),
				attrs),
			_List_fromArray(
				[
					$elm$html$Html$text(value)
				]));
	});
var $author$project$Sharecrop$View$adminPrivacyRequestRow = F2(
	function (resolutionNote, request) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2 py-3 text-sm'),
					$author$project$Sharecrop$Ui$testId('admin-privacy-request')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$badge(request.status),
							A2(
							$elm$html$Html$span,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('font-medium text-slate-900')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(request.kind)
								])),
							A2(
							$elm$html$Html$span,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('break-all text-xs text-slate-500')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(request.id)
								]))
						])),
					A2(
					$elm$html$Html$dl,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('grid gap-2 sm:grid-cols-2')
						]),
					_List_fromArray(
						[
							A2($author$project$Sharecrop$View$operationFact, 'Requested by', request.requestedBy),
							A2($author$project$Sharecrop$View$operationFact, 'Created', request.createdAt),
							A2(
							$author$project$Sharecrop$View$operationFact,
							'Resolved',
							$author$project$Sharecrop$View$emptyLabel(request.resolvedAt)),
							A2(
							$author$project$Sharecrop$View$operationFact,
							'Redacted fields',
							$elm$core$String$fromInt(request.redactedFieldCount))
						])),
					(request.resolutionNote === '') ? $elm$html$Html$text('') : A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs text-slate-600'),
							$author$project$Sharecrop$Ui$testId('admin-privacy-resolution-note')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(request.resolutionNote)
						])),
					(request.exportJSON === '') ? $elm$html$Html$text('') : A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('admin-privacy-export')
						]),
					request.exportJSON),
					(request.status === 'queued') ? A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$ResolveAdminPrivacyRequestClicked(request.id)),
							$elm$html$Html$Attributes$disabled(
							$elm$core$String$trim(resolutionNote) === ''),
							$author$project$Sharecrop$Ui$testId('admin-resolve-privacy')
						]),
					'Resolve') : $elm$html$Html$text('')
				]));
	});
var $author$project$Sharecrop$View$auditEventRow = function (event) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-1 py-2 text-sm'),
				$author$project$Sharecrop$Ui$testId('admin-audit-event')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('font-medium')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(event.action + (' on ' + event.subjectKind))
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-500 break-words')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Subject ' + (event.subjectID + (' · actor ' + (event.actorUserID + (' · ' + event.createdAt)))))
					])),
				(event.metadataJSON === '{}') ? $elm$html$Html$text('') : A2(
				$author$project$Sharecrop$Ui$codeBlock,
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$testId('admin-audit-metadata')
					]),
				event.metadataJSON)
			]));
};
var $elm$html$Html$option = _VirtualDom_node('option');
var $author$project$Sharecrop$View$blankOption = function (labelText) {
	return A2(
		$elm$html$Html$option,
		_List_fromArray(
			[
				A2($elm$html$Html$Attributes$attribute, 'value', '')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(labelText)
			]));
};
var $author$project$Sharecrop$Ui$card = function (children) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-4 shadow-sm sm:p-6')
			]),
		children);
};
var $elm$virtual_dom$VirtualDom$node = function (tag) {
	return _VirtualDom_node(
		_VirtualDom_noScript(tag));
};
var $elm$html$Html$node = $elm$virtual_dom$VirtualDom$node;
var $author$project$Sharecrop$Ui$disclosure = F4(
	function (identifier, openByDefault, title, children) {
		return A3(
			$elm$html$Html$node,
			'details',
			openByDefault ? _List_fromArray(
				[
					A2($elm$html$Html$Attributes$attribute, 'open', '')
				]) : _List_Nil,
			_List_fromArray(
				[
					A3(
					$elm$html$Html$node,
					'summary',
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('cursor-pointer select-none text-lg font-medium marker:text-slate-400'),
							$author$project$Sharecrop$Ui$testId(identifier)
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(title)
						])),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('mt-3 space-y-4')
						]),
					children)
				]));
	});
var $elm$html$Html$label = _VirtualDom_node('label');
var $author$project$Sharecrop$Ui$fieldLabel = F2(
	function (labelText, controls) {
		return A2(
			$elm$html$Html$label,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('block min-w-0 grow space-y-1 text-sm font-medium text-slate-700')
				]),
			A2(
				$elm$core$List$cons,
				A2(
					$elm$html$Html$span,
					_List_Nil,
					_List_fromArray(
						[
							$elm$html$Html$text(labelText)
						])),
				controls));
	});
var $author$project$Sharecrop$Ui$noteText = F2(
	function (identifier, message) {
		return A2(
			$elm$html$Html$p,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-sm text-slate-600'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(message)
				]));
	});
var $author$project$Sharecrop$View$maybeNote = F2(
	function (message, identifier) {
		if (message.$ === 'Just') {
			var value = message.a;
			return A2($author$project$Sharecrop$Ui$noteText, identifier, value);
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$View$paginationControls = F4(
	function (identifier, previous, next, offset) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2 text-xs text-slate-600'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Attributes$disabled(!offset),
							$elm$html$Html$Events$onClick(previous),
							$author$project$Sharecrop$Ui$testId(identifier + '-previous')
						]),
					'Previous'),
					A2(
					$elm$html$Html$span,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId(identifier + '-offset')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(
							'Offset ' + $elm$core$String$fromInt(offset))
						])),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(next),
							$author$project$Sharecrop$Ui$testId(identifier + '-next')
						]),
					'Next')
				]));
	});
var $author$project$Sharecrop$Types$RevokePlatformAdminClicked = function (a) {
	return {$: 'RevokePlatformAdminClicked', a: a};
};
var $author$project$Sharecrop$View$platformAdminRow = function (admin) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-3 py-3 text-sm'),
				$author$project$Sharecrop$Ui$testId('admin-platform-admin')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-1')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('font-medium text-slate-900 break-all')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(admin.userID)
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-500')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(admin.source + (' · ' + admin.createdAt))
							]))
					])),
				(admin.source === 'bootstrap') ? $author$project$Sharecrop$Ui$badge('bootstrap') : A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Sharecrop$Types$RevokePlatformAdminClicked(admin.userID)),
						$author$project$Sharecrop$Ui$testId('admin-revoke-platform-admin')
					]),
				'Revoke')
			]));
};
var $elm$html$Html$select = _VirtualDom_node('select');
var $elm$html$Html$Attributes$selected = $elm$html$Html$Attributes$boolProperty('selected');
var $author$project$Sharecrop$View$stringOption = F2(
	function (selectedValue, _v0) {
		var optionValue = _v0.a;
		var labelText = _v0.b;
		return A2(
			$elm$html$Html$option,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$value(optionValue),
					$elm$html$Html$Attributes$selected(
					_Utils_eq(selectedValue, optionValue))
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Sharecrop$Types$NextUserDirectoryPageClicked = {$: 'NextUserDirectoryPageClicked'};
var $author$project$Sharecrop$Types$PreviousUserDirectoryPageClicked = {$: 'PreviousUserDirectoryPageClicked'};
var $author$project$Sharecrop$Types$SearchUserDirectoryClicked = {$: 'SearchUserDirectoryClicked'};
var $author$project$Sharecrop$Types$UserDirectoryQueryChanged = function (a) {
	return {$: 'UserDirectoryQueryChanged', a: a};
};
var $author$project$Sharecrop$View$selectorSearchControls = F8(
	function (identifier, placeholderText, query, queryChange, search, previous, next, offset) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textInput(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('search'),
									$elm$html$Html$Attributes$placeholder(placeholderText),
									$elm$html$Html$Attributes$value(query),
									$elm$html$Html$Events$onInput(queryChange),
									$author$project$Sharecrop$Ui$testId(identifier + '-query')
								])),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(search),
									$author$project$Sharecrop$Ui$testId(identifier + '-search')
								]),
							'Search')
						])),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2 text-xs text-slate-500')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Attributes$disabled(!offset),
									$elm$html$Html$Events$onClick(previous),
									$author$project$Sharecrop$Ui$testId(identifier + '-previous')
								]),
							'Previous'),
							A2(
							$elm$html$Html$span,
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$testId(identifier + '-offset')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(
									'Offset ' + $elm$core$String$fromInt(offset))
								])),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(next),
									$author$project$Sharecrop$Ui$testId(identifier + '-next')
								]),
							'Next')
						]))
				]));
	});
var $author$project$Sharecrop$View$userPicker = F7(
	function (identifier, selectedUserId, query, change, blankLabel, users, offset) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2')
				]),
			_List_fromArray(
				[
					A8($author$project$Sharecrop$View$selectorSearchControls, identifier, 'Search users', query, $author$project$Sharecrop$Types$UserDirectoryQueryChanged, $author$project$Sharecrop$Types$SearchUserDirectoryClicked, $author$project$Sharecrop$Types$PreviousUserDirectoryPageClicked, $author$project$Sharecrop$Types$NextUserDirectoryPageClicked, offset),
					A2(
					$elm$html$Html$select,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
							$elm$html$Html$Attributes$value(selectedUserId),
							$elm$html$Html$Events$onInput(change),
							$author$project$Sharecrop$Ui$testId(identifier)
						]),
					A2(
						$elm$core$List$cons,
						$author$project$Sharecrop$View$blankOption(blankLabel),
						A2(
							$elm$core$List$map,
							function (user) {
								return A2(
									$elm$html$Html$option,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$value(user.id),
											$elm$html$Html$Attributes$selected(
											_Utils_eq(selectedUserId, user.id))
										]),
									_List_fromArray(
										[
											$elm$html$Html$text(user.email)
										]));
							},
							users)))
				]));
	});
var $author$project$Sharecrop$View$adminView = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'admin-section-operations',
				true,
				'Operations',
				_List_fromArray(
					[
						function () {
						var _v0 = state.operations;
						if (_v0.$ === 'Just') {
							var operations = _v0.a;
							return A2(
								$elm$html$Html$dl,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('grid gap-2 text-sm sm:grid-cols-2'),
										$author$project$Sharecrop$Ui$testId('admin-operations')
									]),
								_List_fromArray(
									[
										A2($author$project$Sharecrop$View$operationFact, 'Status', operations.status),
										A2($author$project$Sharecrop$View$operationFact, 'Account token delivery', operations.accountTokenDelivery),
										A2($author$project$Sharecrop$View$operationFact, 'MCP storage', operations.mcpStorage),
										A2($author$project$Sharecrop$View$operationFact, 'Rate limit storage', operations.rateLimitStorage),
										A2($author$project$Sharecrop$View$operationFact, 'Secure cookies', operations.secureCookies),
										A2(
										$author$project$Sharecrop$View$operationFact,
										'Active MCP sessions',
										$elm$core$String$fromInt(operations.activeMCPSessions)),
										A2(
										$author$project$Sharecrop$View$operationFact,
										'IP rate buckets',
										$elm$core$String$fromInt(operations.activeIPRateBuckets)),
										A2(
										$author$project$Sharecrop$View$operationFact,
										'Subject rate buckets',
										$elm$core$String$fromInt(operations.activeSubjectRateBuckets))
									]));
						} else {
							return A2(
								$elm$html$Html$p,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('text-sm text-slate-500'),
										$author$project$Sharecrop$Ui$testId('admin-operations-empty')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Operations status is not loaded.')
									]));
						}
					}()
					])),
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'admin-section-audit',
				false,
				'Audit events',
				_List_fromArray(
					[
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('grid gap-3 sm:grid-cols-3')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$fieldLabel,
								'Action',
								_List_fromArray(
									[
										$author$project$Sharecrop$Ui$textInput(
										_List_fromArray(
											[
												$elm$html$Html$Attributes$placeholder('submission_accepted'),
												$elm$html$Html$Attributes$value(state.auditActionFilter),
												$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AuditActionFilterChanged),
												$author$project$Sharecrop$Ui$testId('admin-audit-action')
											]))
									])),
								A2(
								$author$project$Sharecrop$Ui$fieldLabel,
								'Subject kind',
								_List_fromArray(
									[
										$author$project$Sharecrop$Ui$textInput(
										_List_fromArray(
											[
												$elm$html$Html$Attributes$placeholder('submission'),
												$elm$html$Html$Attributes$value(state.auditSubjectKindFilter),
												$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AuditSubjectKindFilterChanged),
												$author$project$Sharecrop$Ui$testId('admin-audit-subject-kind')
											]))
									])),
								A2(
								$author$project$Sharecrop$Ui$fieldLabel,
								'Subject ID',
								_List_fromArray(
									[
										$author$project$Sharecrop$Ui$textInput(
										_List_fromArray(
											[
												$elm$html$Html$Attributes$placeholder('ID'),
												$elm$html$Html$Attributes$value(state.auditSubjectIDFilter),
												$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AuditSubjectIDFilterChanged),
												$author$project$Sharecrop$Ui$testId('admin-audit-subject-id')
											]))
									]))
							])),
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('button'),
										$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$SearchAuditEventsClicked),
										$author$project$Sharecrop$Ui$testId('admin-audit-search')
									]),
								'Search')
							])),
						A4($author$project$Sharecrop$View$paginationControls, 'admin-audit-page', $author$project$Sharecrop$Types$PreviousAuditEventsPageClicked, $author$project$Sharecrop$Types$NextAuditEventsPageClicked, state.auditEventsOffset),
						$elm$core$List$isEmpty(state.auditEvents) ? A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm text-slate-500'),
								$author$project$Sharecrop$Ui$testId('admin-audit-empty')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('No audit events.')
							])) : A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
								$author$project$Sharecrop$Ui$testId('admin-audit-events')
							]),
						A2($elm$core$List$map, $author$project$Sharecrop$View$auditEventRow, state.auditEvents))
					])),
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'admin-section-platform-admins',
				false,
				'Platform admins',
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Grant user',
						_List_fromArray(
							[
								A7($author$project$Sharecrop$View$userPicker, 'admin-platform-user', state.adminSelectedUserId, state.userDirectoryQuery, $author$project$Sharecrop$Types$AdminSelectedUserChanged, 'Choose user', state.userDirectory, state.userDirectoryOffset)
							])),
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('button'),
										$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$GrantPlatformAdminClicked),
										$elm$html$Html$Attributes$disabled(
										$elm$core$String$trim(state.adminSelectedUserId) === ''),
										$author$project$Sharecrop$Ui$testId('admin-grant-platform-admin')
									]),
								'Grant')
							])),
						A4($author$project$Sharecrop$View$paginationControls, 'admin-platform-admins-page', $author$project$Sharecrop$Types$PreviousPlatformAdminsPageClicked, $author$project$Sharecrop$Types$NextPlatformAdminsPageClicked, state.platformAdminsOffset),
						$elm$core$List$isEmpty(state.platformAdmins) ? A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm text-slate-500'),
								$author$project$Sharecrop$Ui$testId('admin-platform-admins-empty')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('No platform admins.')
							])) : A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
								$author$project$Sharecrop$Ui$testId('admin-platform-admins')
							]),
						A2($elm$core$List$map, $author$project$Sharecrop$View$platformAdminRow, state.platformAdmins))
					])),
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'admin-section-privacy',
				false,
				'Privacy requests',
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Resolution note',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$placeholder('Export generated or fields redacted'),
										$elm$html$Html$Attributes$value(state.adminPrivacyResolutionNote),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AdminPrivacyResolutionNoteChanged),
										$author$project$Sharecrop$Ui$testId('admin-privacy-note')
									]))
							])),
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('button'),
										$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$RunPrivacyRetentionClicked),
										$author$project$Sharecrop$Ui$testId('admin-run-privacy-retention')
									]),
								'Run retention'),
								function () {
								var _v1 = state.adminRetentionRedactedFieldCount;
								if (_v1.$ === 'Just') {
									var count = _v1.a;
									return A2(
										$elm$html$Html$span,
										_List_fromArray(
											[
												$elm$html$Html$Attributes$class('text-xs text-slate-600'),
												$author$project$Sharecrop$Ui$testId('admin-retention-count')
											]),
										_List_fromArray(
											[
												$elm$html$Html$text(
												'Redacted fields: ' + $elm$core$String$fromInt(count))
											]));
								} else {
									return $elm$html$Html$text('');
								}
							}()
							])),
						A4($author$project$Sharecrop$View$paginationControls, 'admin-privacy-page', $author$project$Sharecrop$Types$PreviousAdminPrivacyPageClicked, $author$project$Sharecrop$Types$NextAdminPrivacyPageClicked, state.adminPrivacyOffset),
						$elm$core$List$isEmpty(state.adminPrivacyRequests) ? A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm text-slate-500'),
								$author$project$Sharecrop$Ui$testId('admin-privacy-empty')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('No privacy requests.')
							])) : A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
								$author$project$Sharecrop$Ui$testId('admin-privacy-requests')
							]),
						A2(
							$elm$core$List$map,
							$author$project$Sharecrop$View$adminPrivacyRequestRow(state.adminPrivacyResolutionNote),
							state.adminPrivacyRequests))
					])),
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'admin-section-moderation',
				false,
				'Moderation reports',
				_List_fromArray(
					[
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('grid gap-3 sm:grid-cols-2')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$fieldLabel,
								'State',
								_List_fromArray(
									[
										A2(
										$elm$html$Html$select,
										_List_fromArray(
											[
												$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
												$elm$html$Html$Attributes$value(state.adminModerationStateFilter),
												$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AdminModerationStateFilterChanged),
												$author$project$Sharecrop$Ui$testId('admin-moderation-state')
											]),
										_List_fromArray(
											[
												$author$project$Sharecrop$View$blankOption('All states'),
												A2(
												$author$project$Sharecrop$View$stringOption,
												state.adminModerationStateFilter,
												_Utils_Tuple2('open', 'Open')),
												A2(
												$author$project$Sharecrop$View$stringOption,
												state.adminModerationStateFilter,
												_Utils_Tuple2('resolved', 'Resolved')),
												A2(
												$author$project$Sharecrop$View$stringOption,
												state.adminModerationStateFilter,
												_Utils_Tuple2('dismissed', 'Dismissed'))
											]))
									])),
								A2(
								$author$project$Sharecrop$Ui$fieldLabel,
								'Triage note',
								_List_fromArray(
									[
										$author$project$Sharecrop$Ui$textInput(
										_List_fromArray(
											[
												$elm$html$Html$Attributes$placeholder('Decision note'),
												$elm$html$Html$Attributes$value(state.adminModerationResolutionNote),
												$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AdminModerationResolutionNoteChanged),
												$author$project$Sharecrop$Ui$testId('admin-moderation-note')
											]))
									]))
							])),
						A4($author$project$Sharecrop$View$paginationControls, 'admin-moderation-page', $author$project$Sharecrop$Types$PreviousAdminModerationPageClicked, $author$project$Sharecrop$Types$NextAdminModerationPageClicked, state.adminModerationOffset),
						$elm$core$List$isEmpty(state.adminModerationReports) ? A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm text-slate-500'),
								$author$project$Sharecrop$Ui$testId('admin-moderation-empty')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('No moderation reports.')
							])) : A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
								$author$project$Sharecrop$Ui$testId('admin-moderation-reports')
							]),
						A2(
							$elm$core$List$map,
							$author$project$Sharecrop$View$adminModerationReportRow(state.adminModerationResolutionNote),
							state.adminModerationReports))
					])),
				A2($author$project$Sharecrop$View$maybeNote, state.adminMessage, 'admin-message')
			]));
};
var $author$project$Sharecrop$Types$AgentLabelChanged = function (a) {
	return {$: 'AgentLabelChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateAgentClicked = {$: 'CreateAgentClicked'};
var $author$project$Sharecrop$Labels$allScopes = _List_fromArray(
	[$author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead, $author$project$Sharecrop$Generated$Agent$AgentScopeTasksWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsRead, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsReview]);
var $author$project$Sharecrop$Labels$credentialStateLabel = function (state) {
	if (state.$ === 'AgentCredentialStateActive') {
		return 'active';
	} else {
		return 'revoked';
	}
};
var $author$project$Sharecrop$Types$RevokeClicked = function (a) {
	return {$: 'RevokeClicked', a: a};
};
var $author$project$Sharecrop$View$revokeButton = function (credential) {
	var _v0 = credential.state;
	if (_v0.$ === 'AgentCredentialStateActive') {
		return A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Events$onClick(
					$author$project$Sharecrop$Types$RevokeClicked(credential.id)),
					$author$project$Sharecrop$Ui$testId('revoke-credential')
				]),
			'Revoke');
	} else {
		return A2(
			$elm$html$Html$span,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-xs text-slate-600')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text('revoked')
				]));
	}
};
var $author$project$Sharecrop$Labels$scopeLabel = function (scope) {
	switch (scope.$) {
		case 'AgentScopeTasksRead':
			return 'Read tasks';
		case 'AgentScopeTasksWrite':
			return 'Create tasks';
		case 'AgentScopeSubmissionsWrite':
			return 'Submit work';
		case 'AgentScopeSubmissionsRead':
			return 'Read submissions';
		default:
			return 'Review submissions';
	}
};
var $author$project$Sharecrop$View$credentialRow = function (credential) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between py-2'),
				$author$project$Sharecrop$Ui$testId('credential-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_Nil,
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('font-medium')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(credential.label)
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-500')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								$author$project$Sharecrop$Labels$credentialStateLabel(credential.state) + (' · ' + A2(
									$elm$core$String$join,
									', ',
									A2($elm$core$List$map, $author$project$Sharecrop$Labels$scopeLabel, credential.scopes))))
							]))
					])),
				$author$project$Sharecrop$View$revokeButton(credential)
			]));
};
var $author$project$Sharecrop$View$credentialsList = function (credentials) {
	return $elm$core$List$isEmpty(credentials) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('credentials')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$credentialRow, credentials));
};
var $author$project$Sharecrop$View$mcpConfig = F2(
	function (origin, secret) {
		return '{\n  \"mcpServers\": {\n    \"sharecrop\": {\n      \"url\": \"' + (origin + ('/mcp\",\n      \"headers\": { \"Authorization\": \"Bearer ' + (secret + '\" }\n    }\n  }\n}')));
	});
var $author$project$Sharecrop$View$newCredentialView = F2(
	function (origin, created) {
		if (created.$ === 'Just') {
			var credential = created.a;
			return A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('mt-4 space-y-3 rounded-md bg-slate-50 p-4')
					]),
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$label_('New agent token (shown once)'),
						A2(
						$author$project$Sharecrop$Ui$codeBlock,
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$testId('agent-secret')
							]),
						credential.secret),
						$author$project$Sharecrop$Ui$label_('MCP client configuration'),
						A2(
						$author$project$Sharecrop$Ui$codeBlock,
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$testId('mcp-config')
							]),
						A2($author$project$Sharecrop$View$mcpConfig, origin, credential.secret))
					]));
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$Types$ToggleScope = function (a) {
	return {$: 'ToggleScope', a: a};
};
var $author$project$Sharecrop$Ui$checkboxClass = 'h-4 w-4 rounded border-slate-400 text-slate-900 focus:ring-2 focus:ring-slate-500';
var $elm$html$Html$Attributes$checked = $elm$html$Html$Attributes$boolProperty('checked');
var $elm$html$Html$Events$targetChecked = A2(
	$elm$json$Json$Decode$at,
	_List_fromArray(
		['target', 'checked']),
	$elm$json$Json$Decode$bool);
var $elm$html$Html$Events$onCheck = function (tagger) {
	return A2(
		$elm$html$Html$Events$on,
		'change',
		A2($elm$json$Json$Decode$map, tagger, $elm$html$Html$Events$targetChecked));
};
var $author$project$Sharecrop$Labels$scopeTag = function (scope) {
	switch (scope.$) {
		case 'AgentScopeTasksRead':
			return 'tasks_read';
		case 'AgentScopeTasksWrite':
			return 'tasks_write';
		case 'AgentScopeSubmissionsWrite':
			return 'submissions_write';
		case 'AgentScopeSubmissionsRead':
			return 'submissions_read';
		default:
			return 'submissions_review';
	}
};
var $author$project$Sharecrop$View$scopeCheckbox = F2(
	function (selected, scope) {
		return A2(
			$elm$html$Html$label,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex min-h-[44px] items-center gap-2 text-sm')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$input,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('checkbox'),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$checkboxClass),
							$elm$html$Html$Attributes$checked(
							A2($elm$core$List$member, scope, selected)),
							$elm$html$Html$Events$onCheck(
							function (_v0) {
								return $author$project$Sharecrop$Types$ToggleScope(scope);
							}),
							$author$project$Sharecrop$Ui$testId(
							'scope-' + $author$project$Sharecrop$Labels$scopeTag(scope))
						]),
					_List_Nil),
					A2(
					$elm$html$Html$span,
					_List_Nil,
					_List_fromArray(
						[
							$elm$html$Html$text(
							$author$project$Sharecrop$Labels$scopeLabel(scope) + (' (' + ($author$project$Sharecrop$Labels$scopeTag(scope) + ')')))
						]))
				]));
	});
var $elm$html$Html$h2 = _VirtualDom_node('h2');
var $author$project$Sharecrop$Ui$sectionTitle = function (title) {
	return A2(
		$elm$html$Html$h2,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-lg font-medium')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(title)
			]));
};
var $author$project$Sharecrop$View$agentsView = F2(
	function (origin, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Agent setup'),
					A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-600')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Create a scoped credential for a local MCP agent.')
						])),
					A2(
					$elm$html$Html$form,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('mt-3 space-y-3'),
							$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$CreateAgentClicked)
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textInput(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('text'),
									$elm$html$Html$Attributes$placeholder('Agent label'),
									$elm$html$Html$Attributes$value(state.agentLabel),
									$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AgentLabelChanged),
									$author$project$Sharecrop$Ui$testId('agent-label')
								])),
							A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('space-y-1')
								]),
							A2(
								$elm$core$List$map,
								$author$project$Sharecrop$View$scopeCheckbox(state.agentScopes),
								$author$project$Sharecrop$Labels$allScopes)),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$author$project$Sharecrop$Ui$testId('create-agent')
								]),
							'Create credential'),
							A2($author$project$Sharecrop$View$maybeNote, state.agentMessage, 'agent-message')
						])),
					A2($author$project$Sharecrop$View$newCredentialView, origin, state.newCredential),
					$author$project$Sharecrop$View$credentialsList(state.credentials)
				]));
	});
var $author$project$Sharecrop$Types$TransferCollectibleClicked = function (a) {
	return {$: 'TransferCollectibleClicked', a: a};
};
var $author$project$Sharecrop$Types$TransferRecipientIdChanged = function (a) {
	return {$: 'TransferRecipientIdChanged', a: a};
};
var $author$project$Sharecrop$Labels$collectibleKindLabel = function (kind) {
	switch (kind.$) {
		case 'CollectibleKindUnique':
			return 'Unique';
		case 'CollectibleKindEdition':
			return 'Edition';
		default:
			return 'Badge';
	}
};
var $author$project$Sharecrop$Labels$collectiblePolicyLabel = function (policy) {
	switch (policy.$) {
		case 'CollectibleTransferPolicyNonTransferableExceptPayout':
			return 'Non-transferable except payout';
		case 'CollectibleTransferPolicyTransferableBetweenUsers':
			return 'Transferable between users';
		case 'CollectibleTransferPolicyTransferableWithinOrganization':
			return 'Transferable within organization';
		default:
			return 'Issuer controlled';
	}
};
var $elm$core$Dict$fromList = function (assocs) {
	return A3(
		$elm$core$List$foldl,
		F2(
			function (_v0, dict) {
				var key = _v0.a;
				var value = _v0.b;
				return A3($elm$core$Dict$insert, key, value, dict);
			}),
		$elm$core$Dict$empty,
		assocs);
};
var $author$project$Sharecrop$Sprites$grey = '#9aa0a6';
var $author$project$Sharecrop$Sprites$ink = '#2a2118';
var $author$project$Sharecrop$Sprites$white = '#ffffff';
var $author$project$Sharecrop$Sprites$placeholderPalette = _List_fromArray(
	[
		_Utils_Tuple2(
		_Utils_chr('k'),
		$author$project$Sharecrop$Sprites$ink),
		_Utils_Tuple2(
		_Utils_chr('w'),
		$author$project$Sharecrop$Sprites$white),
		_Utils_Tuple2(
		_Utils_chr('g'),
		$author$project$Sharecrop$Sprites$grey)
	]);
var $author$project$Sharecrop$Sprites$placeholderRows = _List_fromArray(
	['kkkkkkkkkkkk', 'kgggggggggk.', 'kgwwwwwwwgk.', 'kgwwkkkwwgk.', 'kgwkkgkkwgk.', 'kgwwwwkkwgk.', 'kgwwwkkwwgk.', 'kgwwkkwwwgk.', 'kgwwwwwwwgk.', 'kgwwkkwwwgk.', 'kgggggggggk.', 'kkkkkkkkkkkk']);
var $author$project$Sharecrop$Sprites$brownDark = '#5e3a1a';
var $author$project$Sharecrop$Sprites$green = '#3a7d1e';
var $author$project$Sharecrop$Sprites$greenLight = '#7bb661';
var $author$project$Sharecrop$Sprites$red = '#c0392b';
var $author$project$Sharecrop$Sprites$apple = _Utils_Tuple2(
	_List_fromArray(
		['......b.....', '.....b.gg...', '.....b.gGg..', '...kkbkk....', '..krrrwrrk..', '.krrrrrwrrk.', '.krrrrrrrrk.', '.krrrrrrrrk.', '.krrrrrrrrk.', '..krrrrrrk..', '..krrrrrrk..', '...kkrrkk...']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('r'),
			$author$project$Sharecrop$Sprites$red),
			_Utils_Tuple2(
			_Utils_chr('w'),
			'#e88'),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brownDark),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('G'),
			$author$project$Sharecrop$Sprites$greenLight)
		]));
var $author$project$Sharecrop$Sprites$gold = '#f2c14e';
var $author$project$Sharecrop$Sprites$goldDark = '#e8a33d';
var $author$project$Sharecrop$Sprites$beehive = _Utils_Tuple2(
	_List_fromArray(
		['.....kk.....', '...kkooKK...', '..kooooooak.', '.kooooooooak', '.kKKKKKKKKak', '.koooooooook', '.kKKKKKKKKak', '.kooooooooak', '.kKKKkkKKKak', '.koookkooook', '.kKKkkkkKKak', '..kkkkkkkk..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('K'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$brownDark)
		]));
var $author$project$Sharecrop$Sprites$orange = '#e8772e';
var $author$project$Sharecrop$Sprites$orangeDark = '#c25618';
var $author$project$Sharecrop$Sprites$carrot = _Utils_Tuple2(
	_List_fromArray(
		['...g.g.g....', '..gGgGgGg...', '...gkgkg....', '....kkk.....', '...koook....', '...koaok....', '...koook....', '....koak....', '....koak....', '.....kok....', '.....kak....', '......k.....']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$orange),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$orangeDark),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('G'),
			$author$project$Sharecrop$Sprites$greenLight)
		]));
var $author$project$Sharecrop$Sprites$brown = '#8a5a2b';
var $author$project$Sharecrop$Sprites$cornucopia = _Utils_Tuple2(
	_List_fromArray(
		['..........k.', '........kbbk', '.......kbbk.', '......kbbk.r', '....kbbbkrAr', '...kbbbkoArp', '..kbbbkroArp', '.kbbbkAoArpp', '.kbbkrAoArp.', 'kbbkroArp...', 'kbbkkArp....', '.kkkkkk.....']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brown),
			_Utils_Tuple2(
			_Utils_chr('r'),
			$author$project$Sharecrop$Sprites$red),
			_Utils_Tuple2(
			_Utils_chr('A'),
			$author$project$Sharecrop$Sprites$orange),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('p'),
			'#9a4fb0')
		]));
var $author$project$Sharecrop$Sprites$firstHarvestTrophy = _Utils_Tuple2(
	_List_fromArray(
		['kkkkkkkkkkk.', 'koooooooook.', 'h koooooook h', 'hokooooookoh', 'hokooooookoh', '.hokoooookh.', '..kooooook..', '...kooook...', '....koak....', '....koak....', '...kooook...', '..kkkkkkkk..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('h'),
			$author$project$Sharecrop$Sprites$goldDark)
		]));
var $author$project$Sharecrop$Sprites$foundersSeed = _Utils_Tuple2(
	_List_fromArray(
		['....hhhh....', '..hhwwwwhh..', '.hwwwwwwwwh.', '.hwwkkkkwwh.', 'hwwkooookwwh', 'hwkooaaookwh', 'hwkoaaaaokwh', 'hwkooaaookwh', 'hwwkooookwwh', '.hwwkkkkwwh.', '.hhwwwwwwhh.', '..hhhhhhhh..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('w'),
			'#fdeeb0'),
			_Utils_Tuple2(
			_Utils_chr('h'),
			'#fff6cf')
		]));
var $author$project$Sharecrop$Sprites$greenDark = '#245a10';
var $author$project$Sharecrop$Sprites$moon = '#e8e6d0';
var $author$project$Sharecrop$Sprites$nightField = '#1c2b14';
var $author$project$Sharecrop$Sprites$fullMoonHarvest = _Utils_Tuple2(
	_List_fromArray(
		['nnnnnnnnnnnn', 'nnnn.mmm.nnn', 'nn.mmmMmm.nn', 'n.mmmmmmmm.n', 'n.mmMmmmmm.n', 'n.mmmmmMmm.n', 'nn.mmmmmm.nn', 'nnn.mmmm.nnn', 'nnnnnnnnnnnn', 'ffffffffffff', 'fFfFfFfFfFfF', 'FfFfFfFfFfFf']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('n'),
			$author$project$Sharecrop$Sprites$nightField),
			_Utils_Tuple2(
			_Utils_chr('m'),
			$author$project$Sharecrop$Sprites$moon),
			_Utils_Tuple2(
			_Utils_chr('M'),
			$author$project$Sharecrop$Sprites$grey),
			_Utils_Tuple2(
			_Utils_chr('f'),
			$author$project$Sharecrop$Sprites$greenDark),
			_Utils_Tuple2(
			_Utils_chr('F'),
			$author$project$Sharecrop$Sprites$green)
		]));
var $author$project$Sharecrop$Sprites$greyLight = '#cdd1d6';
var $author$project$Sharecrop$Sprites$goldenCombine = _Utils_Tuple2(
	_List_fromArray(
		['............', 'ooo.....kkk.', 'oaao..kkoook', 'oaaokkkooook', 'oaaooooooook', 'kkkkkooooook', 'kwwwkkkkkkkk', 'kwwwkooooook', 'kkkkkkkkkkk.', '.kykk.kykk..', 'kywwyk kywyk', '.kkk...kkk..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('w'),
			$author$project$Sharecrop$Sprites$greyLight),
			_Utils_Tuple2(
			_Utils_chr('y'),
			$author$project$Sharecrop$Sprites$goldDark)
		]));
var $author$project$Sharecrop$Sprites$goldenEgg = _Utils_Tuple2(
	_List_fromArray(
		['....kkkk....', '...kooowk...', '..kooowook..', '.kooowoook..', '.koowooaook.', 'koowoooaaok.', 'kooooooaaok.', 'koooooaaaok.', 'kooooaaaook.', '.kooaaaaok..', '.kkoaaaokk..', '...kkkkk....']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('w'),
			'#fdeeb0')
		]));
var $author$project$Sharecrop$Sprites$goldenSickle = _Utils_Tuple2(
	_List_fromArray(
		['...kkkkkk...', '..koooooak..', '.kookkkkoak.', 'kook....koak', 'koa......koa', 'kk........kk', '...........k', '.......kk...', '.......bbk..', '......kbbk..', '......kbbk..', '......kbbk..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brown)
		]));
var $author$project$Sharecrop$Sprites$harvestStar = _Utils_Tuple2(
	_List_fromArray(
		['......kk........', '.....koak.......', '.....koak.......', '....koooak.....', 'kkkkooooakkkk..', '.koooooooooak..', '..koooooooak...', '...kooooooak...', '...koooooak....', '..kooak.kooak..', '.kooak...kooak.', '.kak......kak..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark)
		]));
var $author$project$Sharecrop$Sprites$amber = '#d9952b';
var $author$project$Sharecrop$Sprites$honeyPot = _Utils_Tuple2(
	_List_fromArray(
		['.......kk...', '.......bk...', '......bbk...', '....kbbbk...', '...koooook..', '..kkkkkkkk..', '..kaaaaaak..', '.kahhhhhhak.', '.kahhhhhhak.', '.kahhhhhhak.', '.kaahhhhaak.', '..kkaaaakk..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brown),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$amber),
			_Utils_Tuple2(
			_Utils_chr('h'),
			'#f0b84a')
		]));
var $author$project$Sharecrop$Sprites$luckyClover = _Utils_Tuple2(
	_List_fromArray(
		['..kk....kk..', '.kGGk..kGGk.', 'kGgGGkkGGgGk', 'kGGGGGGGGGGk', '.kGGGGGGGGk.', '..kkGddGkk..', '...kGddGk...', '.kGGGddGGGk.', 'kGgGGddGGgGk', 'kGGGGddGGGGk', '.kGGkddkGGk.', '..kk.dd.kk..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('G'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$greenLight),
			_Utils_Tuple2(
			_Utils_chr('d'),
			$author$project$Sharecrop$Sprites$greenDark)
		]));
var $author$project$Sharecrop$Sprites$pink = '#e08aa8';
var $author$project$Sharecrop$Sprites$prizeCow = _Utils_Tuple2(
	_List_fromArray(
		['............', '.kk.....kk..', 'kwwk...kwwk.', 'kwwkkkkkwwk.', '.kwwwwwwwwk.', 'kwbbwwwbbwwk', 'kwbbwwwbbwwk', 'kwwwwpwwwwwk', 'kkwwwwwwwwkk', '.kwwkkkkwwk.', '.kkk.kk.kkk.', '............']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('w'),
			$author$project$Sharecrop$Sprites$white),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brownDark),
			_Utils_Tuple2(
			_Utils_chr('p'),
			$author$project$Sharecrop$Sprites$pink)
		]));
var $author$project$Sharecrop$Sprites$pumpkin = _Utils_Tuple2(
	_List_fromArray(
		['......gg....', '.....gGg....', '.....gg.....', '...kkkkkkk..', '..koaoaoaok.', '.koaoaoaoaok', '.koaoaoaoaok', '.koaoaoaoaok', '.koaoaoaoaok', '..koaoaoaok.', '...kkkkkkk..', '...........']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$orange),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$orangeDark),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('G'),
			$author$project$Sharecrop$Sprites$greenLight)
		]));
var $author$project$Sharecrop$Sprites$blue = '#4a90d9';
var $author$project$Sharecrop$Sprites$rainDrop = _Utils_Tuple2(
	_List_fromArray(
		['.....kk.....', '.....bb.....', '....kbbk....', '....bbbb....', '...kbbwbk...', '...bbbwbb...', '..kbbbbbbk..', '..bbwbbbbb..', '..bbwbbbbb..', '..kbbbbbbk..', '...kbbbbk...', '....kkkk....']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$blue),
			_Utils_Tuple2(
			_Utils_chr('w'),
			'#bcdcf5')
		]));
var $author$project$Sharecrop$Sprites$rainbowField = _Utils_Tuple2(
	_List_fromArray(
		['...rrrrrr...', '..rooooor...', '.rooyyyoor..', '.oyygggyyo..', 'oyggbbbggyo.', 'yggbbppbbggy', 'ggbbp..pbbgg', 'gbbp....pbbg', '............', 'GGGGGGGGGGGG', 'GdGdGdGdGdGd', 'dGdGdGdGdGdG']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('r'),
			$author$project$Sharecrop$Sprites$red),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$orange),
			_Utils_Tuple2(
			_Utils_chr('y'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$blue),
			_Utils_Tuple2(
			_Utils_chr('p'),
			'#9a4fb0'),
			_Utils_Tuple2(
			_Utils_chr('G'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('d'),
			$author$project$Sharecrop$Sprites$greenDark)
		]));
var $author$project$Sharecrop$Sprites$redBarn = _Utils_Tuple2(
	_List_fromArray(
		['.....kk.....', '...kkrrkk...', '.kkrrrrrrkk.', 'krrrrrrrrrrk', 'krrrrrrrrrrk', 'krwwrrrrwwrk', 'krwwrrrrwwrk', 'krrrwwwwrrrk', 'krrwkkkkwrrk', 'krrwkwwkwrrk', 'krrwkwwkwrrk', 'kkkwkkkkwkkk']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('r'),
			$author$project$Sharecrop$Sprites$red),
			_Utils_Tuple2(
			_Utils_chr('w'),
			$author$project$Sharecrop$Sprites$white)
		]));
var $author$project$Sharecrop$Sprites$scarecrow = _Utils_Tuple2(
	_List_fromArray(
		['....kkkkk...', '...kaaaaak..', '..kkkkkkkkk.', '....ooooo...', '...okxoxko..', '...ooooooo..', '...okwwwko..', '..o..ggg..o.', 'ooooogggooooo', '...kogggok..', '....ggggg...', '....b...b...']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$brownDark),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('x'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('w'),
			$author$project$Sharecrop$Sprites$white),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$red),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brown)
		]));
var $author$project$Sharecrop$Sprites$seedling = _Utils_Tuple2(
	_List_fromArray(
		['............', '....k..k....', '...kgk.kgk..', '..kgGgkgGgk.', '..kgGgGgGgk.', '...kgkGkgk..', '.....kGk....', '.....kGk....', '..bbbbbbbbb.', '.bsssssssssb', '.bsbsbsbsbsb', '.bbbbbbbbbb.']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$greenLight),
			_Utils_Tuple2(
			_Utils_chr('G'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brownDark),
			_Utils_Tuple2(
			_Utils_chr('s'),
			$author$project$Sharecrop$Sprites$brown)
		]));
var $author$project$Sharecrop$Sprites$silverPlow = _Utils_Tuple2(
	_List_fromArray(
		['............', '........kk..', '.......kgsk.', '......kgssk.', '.kk..kgssk..', 'kssk.kgsk...', 'kgsskgsk....', '.kgsgssk....', '..kgssk.....', '...kgsk.....', '....kk......', '............']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('s'),
			$author$project$Sharecrop$Sprites$white),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$grey)
		]));
var $author$project$Sharecrop$Sprites$sunToken = _Utils_Tuple2(
	_List_fromArray(
		['..r..r..r..r', '...r.r..r.r.', '....kkkkk...', 'r..koooook.r', '..koooooook.', 'r.koooooook.', '..koooooook.', 'r.koooooook.', '..koooooook.', 'r..koooook.r', '....kkkkk...', '..r.r.r.r.r.']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('r'),
			$author$project$Sharecrop$Sprites$goldDark)
		]));
var $author$project$Sharecrop$Sprites$tractor = _Utils_Tuple2(
	_List_fromArray(
		['............', '......kkkk..', '.....kgggk..', '..kkkkgggk..', '.kgggkkkkk..', '.kgggggggk..', 'kkgggggggkk.', 'kykkkkkkkyk.', 'kywkkkkkywk.', '.kywkkkywk..', 'yk.kywyk.ky.', '............']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('g'),
			$author$project$Sharecrop$Sprites$green),
			_Utils_Tuple2(
			_Utils_chr('y'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('w'),
			$author$project$Sharecrop$Sprites$greyLight)
		]));
var $author$project$Sharecrop$Sprites$wheatSheaf = _Utils_Tuple2(
	_List_fromArray(
		['..o..o..o...', '.oao.oao.oa.', '.oao.oao.oa.', '..o..o..o...', '.oao.oao.oa.', '..o.aoa..o..', '...oaoao....', '...oaoao....', '...bbbbb....', '..bbBBBbb...', '...bbbbb....', '..o.....o..']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$gold),
			_Utils_Tuple2(
			_Utils_chr('a'),
			$author$project$Sharecrop$Sprites$goldDark),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brown),
			_Utils_Tuple2(
			_Utils_chr('B'),
			$author$project$Sharecrop$Sprites$brownDark)
		]));
var $author$project$Sharecrop$Sprites$windmill = _Utils_Tuple2(
	_List_fromArray(
		['kw......wk..', '.kww..wwk...', '..kwwwwk....', '...kook.....', '..wwkkww....', '..wwkkww....', '...kook.....', '...kbbk.....', '..kbwwbk....', '..kbwwbk....', '..kbwwbk....', '..kkkkkk....']),
	_List_fromArray(
		[
			_Utils_Tuple2(
			_Utils_chr('k'),
			$author$project$Sharecrop$Sprites$ink),
			_Utils_Tuple2(
			_Utils_chr('w'),
			$author$project$Sharecrop$Sprites$white),
			_Utils_Tuple2(
			_Utils_chr('o'),
			$author$project$Sharecrop$Sprites$red),
			_Utils_Tuple2(
			_Utils_chr('b'),
			$author$project$Sharecrop$Sprites$brown)
		]));
var $author$project$Sharecrop$Sprites$sprite = function (slug) {
	switch (slug) {
		case 'harvest-star':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$harvestStar);
		case 'golden-sickle':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$goldenSickle);
		case 'seedling':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$seedling);
		case 'sun-token':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$sunToken);
		case 'rain-drop':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$rainDrop);
		case 'wheat-sheaf':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$wheatSheaf);
		case 'red-barn':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$redBarn);
		case 'scarecrow':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$scarecrow);
		case 'honey-pot':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$honeyPot);
		case 'pumpkin':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$pumpkin);
		case 'apple':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$apple);
		case 'carrot':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$carrot);
		case 'beehive':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$beehive);
		case 'windmill':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$windmill);
		case 'tractor':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$tractor);
		case 'silver-plow':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$silverPlow);
		case 'golden-egg':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$goldenEgg);
		case 'prize-cow':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$prizeCow);
		case 'lucky-clover':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$luckyClover);
		case 'full-moon-harvest':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$fullMoonHarvest);
		case 'cornucopia':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$cornucopia);
		case 'first-harvest-trophy':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$firstHarvestTrophy);
		case 'founders-seed':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$foundersSeed);
		case 'rainbow-field':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$rainbowField);
		case 'golden-combine':
			return $elm$core$Maybe$Just($author$project$Sharecrop$Sprites$goldenCombine);
		default:
			return $elm$core$Maybe$Nothing;
	}
};
var $elm$core$List$append = F2(
	function (xs, ys) {
		if (!ys.b) {
			return xs;
		} else {
			return A3($elm$core$List$foldr, $elm$core$List$cons, ys, xs);
		}
	});
var $elm$core$List$concat = function (lists) {
	return A3($elm$core$List$foldr, $elm$core$List$append, _List_Nil, lists);
};
var $elm$core$List$concatMap = F2(
	function (f, list) {
		return $elm$core$List$concat(
			A2($elm$core$List$map, f, list));
	});
var $elm$core$List$maximum = function (list) {
	if (list.b) {
		var x = list.a;
		var xs = list.b;
		return $elm$core$Maybe$Just(
			A3($elm$core$List$foldl, $elm$core$Basics$max, x, xs));
	} else {
		return $elm$core$Maybe$Nothing;
	}
};
var $elm$core$List$repeatHelp = F3(
	function (result, n, value) {
		repeatHelp:
		while (true) {
			if (n <= 0) {
				return result;
			} else {
				var $temp$result = A2($elm$core$List$cons, value, result),
					$temp$n = n - 1,
					$temp$value = value;
				result = $temp$result;
				n = $temp$n;
				value = $temp$value;
				continue repeatHelp;
			}
		}
	});
var $elm$core$List$repeat = F2(
	function (n, value) {
		return A3($elm$core$List$repeatHelp, _List_Nil, n, value);
	});
var $elm$core$String$foldr = _String_foldr;
var $elm$core$String$toList = function (string) {
	return A3($elm$core$String$foldr, $elm$core$List$cons, _List_Nil, string);
};
var $author$project$Sharecrop$Sprites$paddedChars = F2(
	function (columns, row) {
		var chars = $elm$core$String$toList(row);
		return _Utils_ap(
			chars,
			A2(
				$elm$core$List$repeat,
				columns - $elm$core$List$length(chars),
				_Utils_chr('.')));
	});
var $elm$virtual_dom$VirtualDom$style = _VirtualDom_style;
var $elm$html$Html$Attributes$style = $elm$virtual_dom$VirtualDom$style;
var $author$project$Sharecrop$Sprites$renderCell = F3(
	function (cellPx, palette, key) {
		var background = function () {
			switch (key.valueOf()) {
				case '.':
					return 'transparent';
				case ' ':
					return 'transparent';
				default:
					return A2(
						$elm$core$Maybe$withDefault,
						'transparent',
						A2($elm$core$Dict$get, key, palette));
			}
		}();
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					A2($elm$html$Html$Attributes$style, 'width', cellPx),
					A2($elm$html$Html$Attributes$style, 'height', cellPx),
					A2($elm$html$Html$Attributes$style, 'background-color', background)
				]),
			_List_Nil);
	});
var $author$project$Sharecrop$Sprites$spriteFrom = F3(
	function (cell, rows, palette) {
		var columns = A2(
			$elm$core$Maybe$withDefault,
			0,
			$elm$core$List$maximum(
				A2($elm$core$List$map, $elm$core$String$length, rows)));
		var cellPx = $elm$core$String$fromInt(cell) + 'px';
		var cells = A2(
			$elm$core$List$map,
			A2($author$project$Sharecrop$Sprites$renderCell, cellPx, palette),
			A2(
				$elm$core$List$concatMap,
				$author$project$Sharecrop$Sprites$paddedChars(columns),
				rows));
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					A2($elm$html$Html$Attributes$style, 'display', 'grid'),
					A2(
					$elm$html$Html$Attributes$style,
					'grid-template-columns',
					'repeat(' + ($elm$core$String$fromInt(columns) + (', ' + (cellPx + ')')))),
					A2(
					$elm$html$Html$Attributes$style,
					'width',
					$elm$core$String$fromInt(columns * cell) + 'px'),
					A2(
					$elm$html$Html$Attributes$style,
					'height',
					$elm$core$String$fromInt(
						$elm$core$List$length(rows) * cell) + 'px'),
					A2($elm$html$Html$Attributes$style, 'image-rendering', 'pixelated'),
					A2($elm$html$Html$Attributes$style, 'line-height', '0')
				]),
			cells);
	});
var $author$project$Sharecrop$Sprites$pixel = F2(
	function (slug, cell) {
		var _v0 = $author$project$Sharecrop$Sprites$sprite(slug);
		if (_v0.$ === 'Just') {
			var _v1 = _v0.a;
			var rows = _v1.a;
			var palette = _v1.b;
			return A3(
				$author$project$Sharecrop$Sprites$spriteFrom,
				cell,
				rows,
				$elm$core$Dict$fromList(palette));
		} else {
			return A3(
				$author$project$Sharecrop$Sprites$spriteFrom,
				cell,
				$author$project$Sharecrop$Sprites$placeholderRows,
				$elm$core$Dict$fromList($author$project$Sharecrop$Sprites$placeholderPalette));
		}
	});
var $author$project$Sharecrop$View$collectibleDetailView = F2(
	function (collectibleId, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/collectibles'),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
							$author$project$Sharecrop$Ui$testId('back-collectibles')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Back to collectibles')
						])),
					function () {
					var _v0 = A2(
						$elm$core$List$filter,
						function (collectible) {
							return _Utils_eq(collectible.id, collectibleId);
						},
						state.collectibles);
					if (_v0.b) {
						var collectible = _v0.a;
						return A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('mt-3 space-y-2'),
									$author$project$Sharecrop$Ui$testId('collectible-detail')
								]),
							_List_fromArray(
								[
									A2($author$project$Sharecrop$Sprites$pixel, collectible.art, 10),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-2xl font-semibold'),
											$author$project$Sharecrop$Ui$testId('collectible-detail-name')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text(collectible.name)
										])),
									$author$project$Sharecrop$Ui$label_('Collectible ' + collectible.id),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text(
											'Kind: ' + $author$project$Sharecrop$Labels$collectibleKindLabel(collectible.kind))
										])),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text(
											'State: ' + $author$project$Sharecrop$Labels$collectibleStateLabel(collectible.state))
										])),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text(
											'Transfer policy: ' + $author$project$Sharecrop$Labels$collectiblePolicyLabel(collectible.transferPolicy))
										])),
									A2(
									$elm$html$Html$div,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('mt-3 space-y-2')
										]),
									_List_fromArray(
										[
											$author$project$Sharecrop$Ui$label_('Trade to another user'),
											A7($author$project$Sharecrop$View$userPicker, 'transfer-recipient-id', state.transferRecipientId, state.userDirectoryQuery, $author$project$Sharecrop$Types$TransferRecipientIdChanged, 'Choose user', state.userDirectory, state.userDirectoryOffset),
											A2(
											$author$project$Sharecrop$Ui$primaryButton,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$type_('button'),
													$elm$html$Html$Events$onClick(
													$author$project$Sharecrop$Types$TransferCollectibleClicked(collectible.id)),
													$author$project$Sharecrop$Ui$testId('transfer-collectible')
												]),
											'Trade')
										]))
								]));
					} else {
						return A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('mt-3 text-sm text-slate-500'),
									$author$project$Sharecrop$Ui$testId('collectible-detail-missing')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text('This collectible is no longer in your holdings.')
								]));
					}
				}(),
					A2($author$project$Sharecrop$View$maybeNote, state.transferMessage, 'transfer-message')
				]));
	});
var $author$project$Sharecrop$Types$AwardTaskIdChanged = function (a) {
	return {$: 'AwardTaskIdChanged', a: a};
};
var $author$project$Sharecrop$Labels$collectibleCountLabel = function (count) {
	return (count === 1) ? '1 collectible' : ($elm$core$String$fromInt(count) + ' collectibles');
};
var $author$project$Sharecrop$Labels$rewardLabel = F3(
	function (kind, amount, collectibleCount) {
		switch (kind) {
			case 'credit':
				return (collectibleCount > 0) ? ($elm$core$String$fromInt(amount) + (' credits + ' + $author$project$Sharecrop$Labels$collectibleCountLabel(collectibleCount))) : ($elm$core$String$fromInt(amount) + ' credits');
			case 'collectible':
				return $author$project$Sharecrop$Labels$collectibleCountLabel(collectibleCount);
			case 'bundle':
				return $elm$core$String$fromInt(amount) + (' credits + ' + $author$project$Sharecrop$Labels$collectibleCountLabel(collectibleCount));
			default:
				return (collectibleCount > 0) ? $author$project$Sharecrop$Labels$collectibleCountLabel(collectibleCount) : 'no reward';
		}
	});
var $author$project$Sharecrop$Labels$taskStateLabel = function (state) {
	switch (state.$) {
		case 'TaskStateDraft':
			return 'draft';
		case 'TaskStateOpen':
			return 'open';
		case 'TaskStateClosed':
			return 'closed';
		case 'TaskStateCancelled':
			return 'cancelled';
		default:
			return 'expired';
	}
};
var $author$project$Sharecrop$View$taskOption = F2(
	function (selectedTaskId, item) {
		return A2(
			$elm$html$Html$option,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$value(item.id),
					$elm$html$Html$Attributes$selected(
					_Utils_eq(selectedTaskId, item.id))
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(
					item.title + (' · ' + ($author$project$Sharecrop$Labels$taskStateLabel(item.state) + (' · ' + A3($author$project$Sharecrop$Labels$rewardLabel, item.rewardKind, item.rewardCreditAmount, item.rewardCollectibleCount)))))
				]));
	});
var $author$project$Sharecrop$View$taskPicker = F4(
	function (identifier, selectedTaskId, change, tasks) {
		return A2(
			$elm$html$Html$select,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
					$elm$html$Html$Attributes$value(selectedTaskId),
					$elm$html$Html$Events$onInput(change),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			A2(
				$elm$core$List$cons,
				$author$project$Sharecrop$View$blankOption('Select task'),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$taskOption(selectedTaskId),
					tasks)));
	});
var $author$project$Sharecrop$View$awardForm = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 space-y-3')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_('Award a collectible to a task'),
				A4($author$project$Sharecrop$View$taskPicker, 'award-task-id', state.awardTaskId, $author$project$Sharecrop$Types$AwardTaskIdChanged, state.tasks),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-500')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Choose the task here, then press Award next to a collectible below.')
					])),
				A2($author$project$Sharecrop$View$maybeNote, state.awardMessage, 'award-message')
			]));
};
var $author$project$Sharecrop$Types$AwardRecipientKindChanged = function (a) {
	return {$: 'AwardRecipientKindChanged', a: a};
};
var $author$project$Sharecrop$Types$AwardRecipientIdChanged = function (a) {
	return {$: 'AwardRecipientIdChanged', a: a};
};
var $author$project$Sharecrop$Types$NextOrganizationsPageClicked = {$: 'NextOrganizationsPageClicked'};
var $author$project$Sharecrop$Types$NextStandaloneTeamsPageClicked = {$: 'NextStandaloneTeamsPageClicked'};
var $author$project$Sharecrop$Types$OrganizationQueryChanged = function (a) {
	return {$: 'OrganizationQueryChanged', a: a};
};
var $author$project$Sharecrop$Types$PreviousOrganizationsPageClicked = {$: 'PreviousOrganizationsPageClicked'};
var $author$project$Sharecrop$Types$PreviousStandaloneTeamsPageClicked = {$: 'PreviousStandaloneTeamsPageClicked'};
var $author$project$Sharecrop$Types$SearchOrganizationsClicked = {$: 'SearchOrganizationsClicked'};
var $author$project$Sharecrop$Types$SearchStandaloneTeamsClicked = {$: 'SearchStandaloneTeamsClicked'};
var $author$project$Sharecrop$Types$StandaloneTeamQueryChanged = function (a) {
	return {$: 'StandaloneTeamQueryChanged', a: a};
};
var $author$project$Sharecrop$View$organizationPicker = function (identifier) {
	return function (selectedOrganizationId) {
		return function (query) {
			return function (change) {
				return function (queryChange) {
					return function (search) {
						return function (previous) {
							return function (next) {
								return function (blankLabel) {
									return function (organizations) {
										return function (offset) {
											return A2(
												$elm$html$Html$div,
												_List_fromArray(
													[
														$elm$html$Html$Attributes$class('space-y-2')
													]),
												_List_fromArray(
													[
														A8($author$project$Sharecrop$View$selectorSearchControls, identifier, 'Search organizations', query, queryChange, search, previous, next, offset),
														A2(
														$elm$html$Html$select,
														_List_fromArray(
															[
																$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
																$elm$html$Html$Attributes$value(selectedOrganizationId),
																$elm$html$Html$Events$onInput(change),
																$author$project$Sharecrop$Ui$testId(identifier)
															]),
														A2(
															$elm$core$List$cons,
															$author$project$Sharecrop$View$blankOption(blankLabel),
															A2(
																$elm$core$List$map,
																function (organization) {
																	return A2(
																		$elm$html$Html$option,
																		_List_fromArray(
																			[
																				$elm$html$Html$Attributes$value(organization.id),
																				$elm$html$Html$Attributes$selected(
																				_Utils_eq(selectedOrganizationId, organization.id))
																			]),
																		_List_fromArray(
																			[
																				$elm$html$Html$text(organization.name)
																			]));
																},
																organizations)))
													]));
										};
									};
								};
							};
						};
					};
				};
			};
		};
	};
};
var $author$project$Sharecrop$View$teamPicker = function (identifier) {
	return function (selectedTeamId) {
		return function (query) {
			return function (change) {
				return function (queryChange) {
					return function (search) {
						return function (previous) {
							return function (next) {
								return function (blankLabel) {
									return function (teams) {
										return function (offset) {
											return A2(
												$elm$html$Html$div,
												_List_fromArray(
													[
														$elm$html$Html$Attributes$class('space-y-2')
													]),
												_List_fromArray(
													[
														A8($author$project$Sharecrop$View$selectorSearchControls, identifier, 'Search teams', query, queryChange, search, previous, next, offset),
														A2(
														$elm$html$Html$select,
														_List_fromArray(
															[
																$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
																$elm$html$Html$Attributes$value(selectedTeamId),
																$elm$html$Html$Events$onInput(change),
																$author$project$Sharecrop$Ui$testId(identifier)
															]),
														A2(
															$elm$core$List$cons,
															$author$project$Sharecrop$View$blankOption(blankLabel),
															A2(
																$elm$core$List$map,
																function (team) {
																	return A2(
																		$elm$html$Html$option,
																		_List_fromArray(
																			[
																				$elm$html$Html$Attributes$value(team.id),
																				$elm$html$Html$Attributes$selected(
																				_Utils_eq(selectedTeamId, team.id))
																			]),
																		_List_fromArray(
																			[
																				$elm$html$Html$text(team.name)
																			]));
																},
																teams)))
													]));
										};
									};
								};
							};
						};
					};
				};
			};
		};
	};
};
var $author$project$Sharecrop$View$awardRecipientPicker = function (state) {
	return (state.awardRecipientKind === 'organization') ? $author$project$Sharecrop$View$organizationPicker('award-recipient-id')(state.awardRecipientId)(state.organizationQuery)($author$project$Sharecrop$Types$AwardRecipientIdChanged)($author$project$Sharecrop$Types$OrganizationQueryChanged)($author$project$Sharecrop$Types$SearchOrganizationsClicked)($author$project$Sharecrop$Types$PreviousOrganizationsPageClicked)($author$project$Sharecrop$Types$NextOrganizationsPageClicked)('Choose organization')(state.organizations)(state.organizationOffset) : ((state.awardRecipientKind === 'team') ? $author$project$Sharecrop$View$teamPicker('award-recipient-id')(state.awardRecipientId)(state.standaloneTeamQuery)($author$project$Sharecrop$Types$AwardRecipientIdChanged)($author$project$Sharecrop$Types$StandaloneTeamQueryChanged)($author$project$Sharecrop$Types$SearchStandaloneTeamsClicked)($author$project$Sharecrop$Types$PreviousStandaloneTeamsPageClicked)($author$project$Sharecrop$Types$NextStandaloneTeamsPageClicked)('Choose team')(state.standaloneTeams)(state.standaloneTeamOffset) : A7($author$project$Sharecrop$View$userPicker, 'award-recipient-id', state.awardRecipientId, state.userDirectoryQuery, $author$project$Sharecrop$Types$AwardRecipientIdChanged, 'Choose user', state.userDirectory, state.userDirectoryOffset));
};
var $author$project$Sharecrop$View$chooserButton = F4(
	function (isSelected, msg, identifier, labelText) {
		return isSelected ? A2(
			$author$project$Sharecrop$Ui$primaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(msg),
					A2($elm$html$Html$Attributes$attribute, 'aria-pressed', 'true'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			labelText) : A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(msg),
					A2($elm$html$Html$Attributes$attribute, 'aria-pressed', 'false'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			labelText);
	});
var $author$project$Sharecrop$View$awardRecipientControl = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 space-y-3')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_('Admin: award a default collectible'),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-600'),
						$author$project$Sharecrop$Ui$testId('award-admin-note')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Awarding default collectibles requires a platform administrator (enabled in the demo).')
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				_List_fromArray(
					[
						A4(
						$author$project$Sharecrop$View$chooserButton,
						state.awardRecipientKind === 'user',
						$author$project$Sharecrop$Types$AwardRecipientKindChanged('user'),
						'award-kind-user',
						'User'),
						A4(
						$author$project$Sharecrop$View$chooserButton,
						state.awardRecipientKind === 'team',
						$author$project$Sharecrop$Types$AwardRecipientKindChanged('team'),
						'award-kind-team',
						'Team'),
						A4(
						$author$project$Sharecrop$View$chooserButton,
						state.awardRecipientKind === 'organization',
						$author$project$Sharecrop$Types$AwardRecipientKindChanged('organization'),
						'award-kind-organization',
						'Organization')
					])),
				$author$project$Sharecrop$View$awardRecipientPicker(state),
				A2($author$project$Sharecrop$View$maybeNote, state.awardDefaultMessage, 'award-default-message')
			]));
};
var $author$project$Sharecrop$Types$AwardDefaultClicked = function (a) {
	return {$: 'AwardDefaultClicked', a: a};
};
var $author$project$Sharecrop$View$catalogEntry = F3(
	function (isAdmin, recipientId, entry) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-col items-center gap-1 rounded-md border border-slate-200 p-2 text-center'),
					$author$project$Sharecrop$Ui$testId('catalog-entry')
				]),
			_List_fromArray(
				[
					A2($author$project$Sharecrop$Sprites$pixel, entry.art, 6),
					A2(
					$elm$html$Html$span,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs font-medium break-words')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(entry.name)
						])),
					$author$project$Sharecrop$Ui$badge(
					$author$project$Sharecrop$Labels$collectibleKindLabel(entry.kind)),
					isAdmin ? A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$AwardDefaultClicked(entry.slug)),
							$elm$html$Html$Attributes$disabled(
							$elm$core$String$trim(recipientId) === ''),
							$author$project$Sharecrop$Ui$testId('catalog-award')
						]),
					'Award') : $elm$html$Html$text('')
				]));
	});
var $author$project$Sharecrop$View$catalogGallery = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3'),
				$author$project$Sharecrop$Ui$testId('catalog')
			]),
		A2(
			$elm$core$List$map,
			A2($author$project$Sharecrop$View$catalogEntry, state.isAdmin, state.awardRecipientId),
			state.collectibleCatalog));
};
var $author$project$Sharecrop$Types$AwardClicked = function (a) {
	return {$: 'AwardClicked', a: a};
};
var $author$project$Sharecrop$View$awardCollectibleButton = F2(
	function (awardTaskId, collectible) {
		var _v0 = collectible.state;
		if (_v0.$ === 'CollectibleStateMinted') {
			return A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Sharecrop$Types$AwardClicked(collectible.id)),
						$elm$html$Html$Attributes$disabled(awardTaskId === ''),
						$author$project$Sharecrop$Ui$testId('award-collectible')
					]),
				'Award to selected task');
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$View$collectibleRow = F2(
	function (awardTaskId, collectible) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-2 py-2'),
					$author$project$Sharecrop$Ui$testId('collectible-row')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex min-w-0 flex-wrap items-center gap-2')
						]),
					_List_fromArray(
						[
							A2($author$project$Sharecrop$Sprites$pixel, collectible.art, 5),
							A2(
							$elm$html$Html$a,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$href('#/collectibles/' + collectible.id),
									$elm$html$Html$Attributes$class('font-medium underline break-words'),
									$author$project$Sharecrop$Ui$testId('collectible-link')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(collectible.name)
								])),
							$author$project$Sharecrop$Ui$badge(
							$author$project$Sharecrop$Labels$collectibleStateLabel(collectible.state)),
							A2(
							$elm$html$Html$span,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-xs text-slate-500')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(
									$author$project$Sharecrop$Labels$collectibleKindLabel(collectible.kind))
								]))
						])),
					A2($author$project$Sharecrop$View$awardCollectibleButton, awardTaskId, collectible)
				]));
	});
var $author$project$Sharecrop$View$collectiblesList = function (state) {
	return $elm$core$List$isEmpty(state.collectibles) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('collectibles-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('No collectibles yet.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('collectibles')
			]),
		A2(
			$elm$core$List$map,
			$author$project$Sharecrop$View$collectibleRow(state.awardTaskId),
			state.collectibles));
};
var $author$project$Sharecrop$Types$CollectibleNameChanged = function (a) {
	return {$: 'CollectibleNameChanged', a: a};
};
var $author$project$Sharecrop$Types$MintClicked = {$: 'MintClicked'};
var $author$project$Sharecrop$View$allKinds = _List_fromArray(
	[$author$project$Sharecrop$Generated$Collectible$CollectibleKindUnique, $author$project$Sharecrop$Generated$Collectible$CollectibleKindEdition, $author$project$Sharecrop$Generated$Collectible$CollectibleKindBadge]);
var $author$project$Sharecrop$View$allPolicies = _List_fromArray(
	[$author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyNonTransferableExceptPayout, $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableBetweenUsers, $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableWithinOrganization, $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyIssuerControlled]);
var $author$project$Sharecrop$Types$CollectibleKindChosen = function (a) {
	return {$: 'CollectibleKindChosen', a: a};
};
var $author$project$Sharecrop$Labels$collectibleKindTag = function (kind) {
	switch (kind.$) {
		case 'CollectibleKindUnique':
			return 'unique';
		case 'CollectibleKindEdition':
			return 'edition';
		default:
			return 'badge';
	}
};
var $author$project$Sharecrop$View$kindButton = F2(
	function (selected, kind) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, kind),
			$author$project$Sharecrop$Types$CollectibleKindChosen(kind),
			'collectible-kind-' + $author$project$Sharecrop$Labels$collectibleKindTag(kind),
			$author$project$Sharecrop$Labels$collectibleKindLabel(kind));
	});
var $author$project$Sharecrop$Types$CollectiblePolicyChosen = function (a) {
	return {$: 'CollectiblePolicyChosen', a: a};
};
var $author$project$Sharecrop$Labels$collectiblePolicyTag = function (policy) {
	switch (policy.$) {
		case 'CollectibleTransferPolicyNonTransferableExceptPayout':
			return 'non_transferable_except_payout';
		case 'CollectibleTransferPolicyTransferableBetweenUsers':
			return 'transferable_between_users';
		case 'CollectibleTransferPolicyTransferableWithinOrganization':
			return 'transferable_within_organization';
		default:
			return 'issuer_controlled';
	}
};
var $author$project$Sharecrop$View$policyButton = F2(
	function (selected, policy) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, policy),
			$author$project$Sharecrop$Types$CollectiblePolicyChosen(policy),
			'collectible-policy-' + $author$project$Sharecrop$Labels$collectiblePolicyTag(policy),
			$author$project$Sharecrop$Labels$collectiblePolicyLabel(policy));
	});
var $author$project$Sharecrop$View$mintForm = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-3 space-y-3'),
				$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$MintClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('text'),
						$elm$html$Html$Attributes$placeholder('Collectible name'),
						$elm$html$Html$Attributes$value(state.collectibleName),
						$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CollectibleNameChanged),
						$author$project$Sharecrop$Ui$testId('collectible-name')
					])),
				$author$project$Sharecrop$Ui$label_('Kind'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$kindButton(state.collectibleKind),
					$author$project$Sharecrop$View$allKinds)),
				$author$project$Sharecrop$Ui$label_('Transfer policy'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$policyButton(state.collectiblePolicy),
					$author$project$Sharecrop$View$allPolicies)),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$author$project$Sharecrop$Ui$testId('mint-collectible')
					]),
				'Mint collectible'),
				A2($author$project$Sharecrop$View$maybeNote, state.collectibleMessage, 'collectible-message')
			]));
};
var $author$project$Sharecrop$View$collectiblesView = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Collectibles'),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-600')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Mint your own collectibles, award default collectibles to users, teams, or organizations, and trade collectibles to other users.')
					])),
				$author$project$Sharecrop$View$mintForm(state),
				$author$project$Sharecrop$View$awardForm(state),
				state.isAdmin ? $author$project$Sharecrop$View$awardRecipientControl(state) : $elm$html$Html$text(''),
				$author$project$Sharecrop$View$catalogGallery(state),
				$author$project$Sharecrop$View$collectiblesList(state)
			]));
};
var $author$project$Sharecrop$Types$CreateDescriptionChanged = function (a) {
	return {$: 'CreateDescriptionChanged', a: a};
};
var $author$project$Sharecrop$Types$CreatePayloadChanged = function (a) {
	return {$: 'CreatePayloadChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateReferenceURLChanged = function (a) {
	return {$: 'CreateReferenceURLChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateReservationHoursChanged = function (a) {
	return {$: 'CreateReservationHoursChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateResponseSchemaChanged = function (a) {
	return {$: 'CreateResponseSchemaChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateTaskClicked = {$: 'CreateTaskClicked'};
var $author$project$Sharecrop$Types$CreateTitleChanged = function (a) {
	return {$: 'CreateTitleChanged', a: a};
};
var $author$project$Sharecrop$Types$PickCreateAttachmentClicked = {$: 'PickCreateAttachmentClicked'};
var $author$project$Sharecrop$Types$RemoveCreateAttachmentClicked = function (a) {
	return {$: 'RemoveCreateAttachmentClicked', a: a};
};
var $author$project$Sharecrop$View$allAssigneeScopes = _List_fromArray(
	[$author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser, $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeOrganizationTeam, $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeTeam]);
var $author$project$Sharecrop$View$allParticipationPolicies = _List_fromArray(
	[$author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen, $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyReservationRequired, $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyApprovalRequired]);
var $author$project$Sharecrop$View$allRewardKinds = _List_fromArray(
	['none', 'credit', 'collectible', 'bundle']);
var $author$project$Sharecrop$Types$visibilityPublicTag = 'public';
var $author$project$Sharecrop$Types$allVisibilityTags = _List_fromArray(
	[$author$project$Sharecrop$Types$visibilityPublicTag, $author$project$Sharecrop$Types$visibilityDefaultTag, $author$project$Sharecrop$Types$visibilityUserTag, $author$project$Sharecrop$Types$visibilityTeamTag, $author$project$Sharecrop$Types$visibilityOrganizationTag]);
var $author$project$Sharecrop$Types$CreateAssigneeScopeChosen = function (a) {
	return {$: 'CreateAssigneeScopeChosen', a: a};
};
var $author$project$Sharecrop$Labels$assigneeScopeLabel = function (scope) {
	switch (scope.$) {
		case 'TaskAssigneeScopeUser':
			return 'user';
		case 'TaskAssigneeScopeOrganizationTeam':
			return 'organization team';
		default:
			return 'team';
	}
};
var $author$project$Sharecrop$View$assigneeScopeButton = F2(
	function (selected, scope) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, scope),
			$author$project$Sharecrop$Types$CreateAssigneeScopeChosen(scope),
			'create-assignee-' + $author$project$Sharecrop$Labels$assigneeScopeTag(scope),
			$author$project$Sharecrop$Labels$assigneeScopeLabel(scope));
	});
var $author$project$Sharecrop$Types$CreateTaskOwnerChanged = function (a) {
	return {$: 'CreateTaskOwnerChanged', a: a};
};
var $author$project$Sharecrop$View$ownerButton = F2(
	function (selected, organization) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, organization.id),
			$author$project$Sharecrop$Types$CreateTaskOwnerChanged(organization.id),
			'create-owner-' + organization.id,
			organization.name);
	});
var $author$project$Sharecrop$View$ownerChooser = function (state) {
	return $elm$core$List$isEmpty(state.organizations) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_Nil,
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_('Owner'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
						$author$project$Sharecrop$Ui$testId('create-owner')
					]),
				A2(
					$elm$core$List$cons,
					A4(
						$author$project$Sharecrop$View$chooserButton,
						state.createTaskOwner === '',
						$author$project$Sharecrop$Types$CreateTaskOwnerChanged(''),
						'create-owner-me',
						'Me'),
					A2(
						$elm$core$List$map,
						$author$project$Sharecrop$View$ownerButton(state.createTaskOwner),
						state.organizations)))
			]));
};
var $author$project$Sharecrop$Types$CreateParticipationChanged = function (a) {
	return {$: 'CreateParticipationChanged', a: a};
};
var $author$project$Sharecrop$Labels$participationPolicyLabel = function (policy) {
	switch (policy.$) {
		case 'TaskParticipationPolicyOpen':
			return 'open submissions';
		case 'TaskParticipationPolicyReservationRequired':
			return 'reservation required';
		default:
			return 'approval required';
	}
};
var $author$project$Sharecrop$View$participationButton = F2(
	function (selectedPolicy, policy) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(
				selectedPolicy,
				$author$project$Sharecrop$Labels$participationPolicyTag(policy)),
			$author$project$Sharecrop$Types$CreateParticipationChanged(
				$author$project$Sharecrop$Labels$participationPolicyTag(policy)),
			'create-participation-' + $author$project$Sharecrop$Labels$participationPolicyTag(policy),
			$author$project$Sharecrop$Labels$participationPolicyLabel(policy));
	});
var $author$project$Sharecrop$Types$CreateRewardAmountChanged = function (a) {
	return {$: 'CreateRewardAmountChanged', a: a};
};
var $author$project$Sharecrop$View$rewardAmountField = function (state) {
	return ((state.createRewardKind === 'credit') || (state.createRewardKind === 'bundle')) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Credit amount',
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('number'),
						$elm$html$Html$Attributes$placeholder('Amount in credits'),
						$elm$html$Html$Attributes$value(state.createRewardAmount),
						$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateRewardAmountChanged),
						$author$project$Sharecrop$Ui$testId('create-reward')
					]))
			])) : $elm$html$Html$text('');
};
var $author$project$Sharecrop$Types$ToggleCreateRewardCollectible = function (a) {
	return {$: 'ToggleCreateRewardCollectible', a: a};
};
var $author$project$Sharecrop$Ui$checkbox = F2(
	function (attrs, labelText) {
		return A2(
			$elm$html$Html$label,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex items-center gap-2 text-sm text-slate-700')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$input,
					A2(
						$elm$core$List$cons,
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$checkboxClass),
						A2(
							$elm$core$List$cons,
							$elm$html$Html$Attributes$type_('checkbox'),
							attrs)),
					_List_Nil),
					A2(
					$elm$html$Html$span,
					_List_Nil,
					_List_fromArray(
						[
							$elm$html$Html$text(labelText)
						]))
				]));
	});
var $author$project$Sharecrop$View$rewardCollectibleField = function (state) {
	if ((state.createRewardKind === 'collectible') || (state.createRewardKind === 'bundle')) {
		var available = A2(
			$elm$core$List$filter,
			function (collectible) {
				return _Utils_eq(collectible.state, $author$project$Sharecrop$Generated$Collectible$CollectibleStateMinted);
			},
			state.collectibles);
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2'),
					$author$project$Sharecrop$Ui$testId('create-reward-collectibles')
				]),
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$label_('Collectibles'),
					$elm$core$List$isEmpty(available) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No minted collectibles available.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-1')
						]),
					A2(
						$elm$core$List$map,
						function (collectible) {
							return A2(
								$author$project$Sharecrop$Ui$checkbox,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$checked(
										A2($elm$core$List$member, collectible.id, state.createRewardCollectibleIds)),
										$elm$html$Html$Events$onCheck(
										function (_v0) {
											return $author$project$Sharecrop$Types$ToggleCreateRewardCollectible(collectible.id);
										}),
										$author$project$Sharecrop$Ui$testId('create-reward-collectible-' + collectible.id)
									]),
								collectible.name + (' · ' + $author$project$Sharecrop$Labels$collectibleKindLabel(collectible.kind)));
						},
						available))
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$Types$CreateRewardKindChanged = function (a) {
	return {$: 'CreateRewardKindChanged', a: a};
};
var $author$project$Sharecrop$View$rewardKindLabel = function (kind) {
	switch (kind) {
		case 'credit':
			return 'Credits';
		case 'collectible':
			return 'Collectible';
		case 'bundle':
			return 'Bundle';
		default:
			return 'No reward';
	}
};
var $author$project$Sharecrop$View$rewardKindButton = F2(
	function (selected, kind) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, kind),
			$author$project$Sharecrop$Types$CreateRewardKindChanged(kind),
			'create-reward-kind-' + kind,
			$author$project$Sharecrop$View$rewardKindLabel(kind));
	});
var $elm$html$Html$Attributes$rows = function (n) {
	return A2(
		_VirtualDom_attribute,
		'rows',
		$elm$core$String$fromInt(n));
};
var $author$project$Sharecrop$Types$AddSchemaFieldClicked = {$: 'AddSchemaFieldClicked'};
var $author$project$Sharecrop$Types$RemoveSchemaFieldClicked = function (a) {
	return {$: 'RemoveSchemaFieldClicked', a: a};
};
var $author$project$Sharecrop$Types$SchemaFieldKindChanged = F2(
	function (a, b) {
		return {$: 'SchemaFieldKindChanged', a: a, b: b};
	});
var $author$project$Sharecrop$Types$SchemaFieldNameChanged = F2(
	function (a, b) {
		return {$: 'SchemaFieldNameChanged', a: a, b: b};
	});
var $author$project$Sharecrop$Types$SchemaFieldRequiredChanged = F2(
	function (a, b) {
		return {$: 'SchemaFieldRequiredChanged', a: a, b: b};
	});
var $author$project$Sharecrop$Types$SchemaFieldEnumValuesChanged = F2(
	function (a, b) {
		return {$: 'SchemaFieldEnumValuesChanged', a: a, b: b};
	});
var $author$project$Sharecrop$Types$SchemaFieldItemKindChanged = F2(
	function (a, b) {
		return {$: 'SchemaFieldItemKindChanged', a: a, b: b};
	});
var $author$project$Sharecrop$View$schemaItemKinds = _List_fromArray(
	['string', 'integer', 'decimal_string']);
var $author$project$Sharecrop$View$schemaKindOption = F2(
	function (selectedKind, kind) {
		return A2(
			$elm$html$Html$option,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$value(kind),
					$elm$html$Html$Attributes$selected(
					_Utils_eq(kind, selectedKind))
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(kind)
				]));
	});
var $author$project$Sharecrop$View$schemaFieldDetail = F2(
	function (index, field) {
		var _v0 = field.kind;
		switch (_v0) {
			case 'enum':
				return A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('w-full')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Allowed values (comma-separated)',
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$textInput(
									_List_fromArray(
										[
											$elm$html$Html$Attributes$type_('text'),
											$elm$html$Html$Attributes$placeholder('low, medium, high'),
											$elm$html$Html$Attributes$value(field.enumValues),
											$elm$html$Html$Events$onInput(
											$author$project$Sharecrop$Types$SchemaFieldEnumValuesChanged(index)),
											$author$project$Sharecrop$Ui$testId('schema-field-enum-values')
										]))
								]))
						]));
			case 'array':
				return A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('w-full sm:w-auto')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Item type',
							_List_fromArray(
								[
									A2(
									$elm$html$Html$select,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
											$elm$html$Html$Attributes$value(field.itemKind),
											$elm$html$Html$Events$onInput(
											$author$project$Sharecrop$Types$SchemaFieldItemKindChanged(index)),
											$author$project$Sharecrop$Ui$testId('schema-field-item-kind')
										]),
									A2(
										$elm$core$List$map,
										$author$project$Sharecrop$View$schemaKindOption(field.itemKind),
										$author$project$Sharecrop$View$schemaItemKinds))
								]))
						]));
			default:
				return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$View$schemaFieldKinds = _List_fromArray(
	['string', 'integer', 'decimal_string', 'enum', 'array', 'freeform']);
var $author$project$Sharecrop$View$schemaFieldRow = F2(
	function (index, field) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2 rounded-md border border-slate-200 bg-white p-3')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-col gap-2 sm:flex-row sm:items-end')
						]),
					_List_fromArray(
						[
							A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('w-full sm:flex-1')
								]),
							_List_fromArray(
								[
									A2(
									$author$project$Sharecrop$Ui$fieldLabel,
									'Field name',
									_List_fromArray(
										[
											$author$project$Sharecrop$Ui$textInput(
											_List_fromArray(
												[
													$elm$html$Html$Attributes$type_('text'),
													$elm$html$Html$Attributes$placeholder('summary'),
													$elm$html$Html$Attributes$value(field.name),
													$elm$html$Html$Events$onInput(
													$author$project$Sharecrop$Types$SchemaFieldNameChanged(index)),
													$author$project$Sharecrop$Ui$testId('schema-field-name')
												]))
										]))
								])),
							A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('w-full sm:w-auto')
								]),
							_List_fromArray(
								[
									A2(
									$author$project$Sharecrop$Ui$fieldLabel,
									'Type',
									_List_fromArray(
										[
											A2(
											$elm$html$Html$select,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
													$elm$html$Html$Attributes$value(field.kind),
													$elm$html$Html$Events$onInput(
													$author$project$Sharecrop$Types$SchemaFieldKindChanged(index)),
													$author$project$Sharecrop$Ui$testId('schema-field-kind')
												]),
											A2(
												$elm$core$List$map,
												$author$project$Sharecrop$View$schemaKindOption(field.kind),
												$author$project$Sharecrop$View$schemaFieldKinds))
										]))
								])),
							A2(
							$elm$html$Html$label,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('flex min-h-[44px] w-full items-center gap-2 text-sm text-slate-700 sm:w-auto')
								]),
							_List_fromArray(
								[
									A2(
									$elm$html$Html$input,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$type_('checkbox'),
											$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$checkboxClass),
											$elm$html$Html$Attributes$checked(field.required),
											$elm$html$Html$Events$onCheck(
											$author$project$Sharecrop$Types$SchemaFieldRequiredChanged(index)),
											$author$project$Sharecrop$Ui$testId('schema-field-required')
										]),
									_List_Nil),
									$elm$html$Html$text('Required')
								])),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									$author$project$Sharecrop$Types$RemoveSchemaFieldClicked(index)),
									$author$project$Sharecrop$Ui$testId('schema-field-remove'),
									$elm$html$Html$Attributes$class('w-full sm:w-auto')
								]),
							'Remove')
						])),
					A2($author$project$Sharecrop$View$schemaFieldDetail, index, field)
				]));
	});
var $author$project$Sharecrop$View$schemaDesignerView = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-3 rounded-md border border-slate-200 bg-slate-50 p-4')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_('Response schema designer'),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-600')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Add fields to build an object schema without writing JSON. Pick a type per field — enum and array prompt for their values. With no fields the schema is freeform.')
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2')
					]),
				A2($elm$core$List$indexedMap, $author$project$Sharecrop$View$schemaFieldRow, state.createSchemaFields)),
				A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$AddSchemaFieldClicked),
						$author$project$Sharecrop$Ui$testId('schema-add-field')
					]),
				'Add field')
			]));
};
var $author$project$Sharecrop$View$selectedAttachmentRow = F3(
	function (removeMsg, index, attachment) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-2 rounded border border-slate-200 px-3 py-2 text-sm'),
					$author$project$Sharecrop$Ui$testId('selected-attachment')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$span,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('break-all text-slate-700')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(
							attachment.name + (' · ' + (attachment.contentType + (' · ' + ($elm$core$String$fromInt(attachment.sizeBytes) + ' bytes')))))
						])),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							removeMsg(index)),
							$author$project$Sharecrop$Ui$testId('remove-attachment')
						]),
					'Remove')
				]));
	});
var $author$project$Sharecrop$View$selectedAttachmentsView = F5(
	function (labelText, attachments, pickMsg, removeMsg, id) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2'),
					$author$project$Sharecrop$Ui$testId(id)
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$label_(labelText),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(pickMsg),
									$author$project$Sharecrop$Ui$testId(id + '-pick')
								]),
							'Add file')
						])),
					$elm$core$List$isEmpty(attachments) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs text-slate-500')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No files attached.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-1')
						]),
					A2(
						$elm$core$List$indexedMap,
						$author$project$Sharecrop$View$selectedAttachmentRow(removeMsg),
						attachments))
				]));
	});
var $author$project$Sharecrop$View$taskTypeLabel = function (tag) {
	switch (tag) {
		case 'code_review':
			return 'Code review';
		case 'security_review':
			return 'Security review';
		case 'product_review':
			return 'Product review';
		case 'ui_ux_review':
			return 'UI/UX review';
		case 'qa_testing':
			return 'QA testing';
		default:
			return 'General';
	}
};
var $author$project$Sharecrop$Types$CreateTaskTypeChanged = function (a) {
	return {$: 'CreateTaskTypeChanged', a: a};
};
var $author$project$Sharecrop$View$allTaskTypes = _List_fromArray(
	['general', 'code_review', 'security_review', 'product_review', 'ui_ux_review', 'qa_testing']);
var $author$project$Sharecrop$View$taskTypeOption = F2(
	function (selectedType, tag) {
		var optionLabel = (tag === 'general') ? 'Freeform (no template)' : $author$project$Sharecrop$View$taskTypeLabel(tag);
		return A2(
			$elm$html$Html$option,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$value(tag),
					$elm$html$Html$Attributes$selected(
					_Utils_eq(selectedType, tag))
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(optionLabel)
				]));
	});
var $author$project$Sharecrop$View$taskTypeSelect = function (selectedType) {
	return A2(
		$elm$html$Html$select,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
				$elm$html$Html$Attributes$value(selectedType),
				$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateTaskTypeChanged),
				$author$project$Sharecrop$Ui$testId('create-task-type')
			]),
		A2(
			$elm$core$List$map,
			$author$project$Sharecrop$View$taskTypeOption(selectedType),
			$author$project$Sharecrop$View$allTaskTypes));
};
var $elm$html$Html$textarea = _VirtualDom_node('textarea');
var $author$project$Sharecrop$Ui$textareaClass = 'w-full rounded-md border border-slate-300 px-3 py-2 font-mono text-sm focus:border-slate-500 focus:outline-none';
var $author$project$Sharecrop$Ui$textarea_ = function (attrs) {
	return A2(
		$elm$html$Html$textarea,
		A2(
			$elm$core$List$cons,
			$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$textareaClass),
			attrs),
		_List_Nil);
};
var $author$project$Sharecrop$Types$CreateVisibilityChanged = function (a) {
	return {$: 'CreateVisibilityChanged', a: a};
};
var $author$project$Sharecrop$Types$visibilityLabel = function (tag) {
	return _Utils_eq(tag, $author$project$Sharecrop$Types$visibilityPublicTag) ? 'Public' : (_Utils_eq(tag, $author$project$Sharecrop$Types$visibilityUserTag) ? 'Specific user' : (_Utils_eq(tag, $author$project$Sharecrop$Types$visibilityTeamTag) ? 'Team' : (_Utils_eq(tag, $author$project$Sharecrop$Types$visibilityOrganizationTag) ? 'Organization' : 'Private (default)')));
};
var $author$project$Sharecrop$View$visibilityButton = F2(
	function (selected, tag) {
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, tag),
			$author$project$Sharecrop$Types$CreateVisibilityChanged(tag),
			'create-visibility-' + tag,
			$author$project$Sharecrop$Types$visibilityLabel(tag));
	});
var $author$project$Sharecrop$Types$CreateScopeOrganizationIdChanged = function (a) {
	return {$: 'CreateScopeOrganizationIdChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateScopeTeamIdChanged = function (a) {
	return {$: 'CreateScopeTeamIdChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateScopeUserIdChanged = function (a) {
	return {$: 'CreateScopeUserIdChanged', a: a};
};
var $author$project$Sharecrop$View$visibilityScopeField = function (state) {
	return _Utils_eq(state.createVisibility, $author$project$Sharecrop$Types$visibilityUserTag) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Share with user',
		_List_fromArray(
			[
				A7($author$project$Sharecrop$View$userPicker, 'create-scope-user', state.createScopeUserId, state.userDirectoryQuery, $author$project$Sharecrop$Types$CreateScopeUserIdChanged, 'Choose user', state.userDirectory, state.userDirectoryOffset)
			])) : (_Utils_eq(state.createVisibility, $author$project$Sharecrop$Types$visibilityTeamTag) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Share with team',
		_List_fromArray(
			[
				$author$project$Sharecrop$View$teamPicker('create-scope-team')(state.createScopeTeamId)(state.standaloneTeamQuery)($author$project$Sharecrop$Types$CreateScopeTeamIdChanged)($author$project$Sharecrop$Types$StandaloneTeamQueryChanged)($author$project$Sharecrop$Types$SearchStandaloneTeamsClicked)($author$project$Sharecrop$Types$PreviousStandaloneTeamsPageClicked)($author$project$Sharecrop$Types$NextStandaloneTeamsPageClicked)('Choose team')(state.standaloneTeams)(state.standaloneTeamOffset)
			])) : (_Utils_eq(state.createVisibility, $author$project$Sharecrop$Types$visibilityOrganizationTag) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Share with organization',
		_List_fromArray(
			[
				$author$project$Sharecrop$View$organizationPicker('create-scope-organization')(state.createScopeOrganizationId)(state.organizationQuery)($author$project$Sharecrop$Types$CreateScopeOrganizationIdChanged)($author$project$Sharecrop$Types$OrganizationQueryChanged)($author$project$Sharecrop$Types$SearchOrganizationsClicked)($author$project$Sharecrop$Types$PreviousOrganizationsPageClicked)($author$project$Sharecrop$Types$NextOrganizationsPageClicked)('Choose organization')(state.organizations)(state.organizationOffset)
			])) : $elm$html$Html$text('')));
};
var $author$project$Sharecrop$View$createTaskView = function (state) {
	var advancedActive = (state.createTaskType !== 'general') || (($elm$core$String$trim(state.createReferenceURL) !== '') || (($elm$core$String$trim(state.createPayloadJson) !== '') || (!$elm$core$List$isEmpty(state.createAttachments))));
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$CreateTaskClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Create a task'),
				A2(
				$author$project$Sharecrop$Ui$fieldLabel,
				'Title',
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('text'),
								$elm$html$Html$Attributes$placeholder('Short, descriptive title'),
								$elm$html$Html$Attributes$value(state.createTitle),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateTitleChanged),
								$author$project$Sharecrop$Ui$testId('create-title')
							]))
					])),
				A2(
				$author$project$Sharecrop$Ui$fieldLabel,
				'Template',
				_List_fromArray(
					[
						$author$project$Sharecrop$View$taskTypeSelect(state.createTaskType)
					])),
				A2(
				$author$project$Sharecrop$Ui$fieldLabel,
				'Description',
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$textarea_(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$placeholder('What the worker should do'),
								$elm$html$Html$Attributes$value(state.createDescription),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateDescriptionChanged),
								$elm$html$Html$Attributes$rows(3),
								$author$project$Sharecrop$Ui$testId('create-description')
							]))
					])),
				(state.createTaskType === 'general') ? $author$project$Sharecrop$View$schemaDesignerView(state) : A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-600'),
						$author$project$Sharecrop$Ui$testId('template-schema-note')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(
						'The ' + ($author$project$Sharecrop$View$taskTypeLabel(state.createTaskType) + ' template prefilled the description and response schema; open Advanced options below to review or edit the schema.'))
					])),
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'create-advanced-options',
				advancedActive,
				'Advanced options',
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Reference URL (optional, e.g. a pull request)',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('text'),
										$elm$html$Html$Attributes$placeholder('https://github.com/org/repo/pull/123'),
										$elm$html$Html$Attributes$value(state.createReferenceURL),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateReferenceURLChanged),
										$author$project$Sharecrop$Ui$testId('create-reference-url')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Response schema (JSON, advanced)',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textarea_(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$placeholder('{\"kind\":\"freeform\"}'),
										$elm$html$Html$Attributes$value(state.createResponseSchema),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateResponseSchemaChanged),
										$elm$html$Html$Attributes$rows(3),
										$author$project$Sharecrop$Ui$testId('create-response-schema')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Task input (JSON, optional)',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textarea_(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$placeholder('Embed any data the worker needs, or leave blank'),
										$elm$html$Html$Attributes$value(state.createPayloadJson),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreatePayloadChanged),
										$elm$html$Html$Attributes$rows(3),
										$author$project$Sharecrop$Ui$testId('create-payload')
									]))
							])),
						A5($author$project$Sharecrop$View$selectedAttachmentsView, 'Attachments', state.createAttachments, $author$project$Sharecrop$Types$PickCreateAttachmentClicked, $author$project$Sharecrop$Types$RemoveCreateAttachmentClicked, 'create-attachments')
					])),
				$author$project$Sharecrop$Ui$label_('Reward'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$rewardKindButton(state.createRewardKind),
					$author$project$Sharecrop$View$allRewardKinds)),
				$author$project$Sharecrop$View$rewardAmountField(state),
				$author$project$Sharecrop$View$rewardCollectibleField(state),
				$author$project$Sharecrop$View$ownerChooser(state),
				$author$project$Sharecrop$Ui$label_('Participation'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$participationButton(state.createParticipationPolicy),
					$author$project$Sharecrop$View$allParticipationPolicies)),
				$author$project$Sharecrop$Labels$participationUsesReservation(state.createParticipationPolicy) ? A2(
				$author$project$Sharecrop$Ui$fieldLabel,
				'Reservation expiry (hours)',
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('number'),
								$elm$html$Html$Attributes$placeholder('48'),
								$elm$html$Html$Attributes$value(state.createReservationHours),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateReservationHoursChanged),
								$author$project$Sharecrop$Ui$testId('create-reservation-hours')
							]))
					])) : $elm$html$Html$text(''),
				$author$project$Sharecrop$Ui$label_('Visibility'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$visibilityButton(state.createVisibility),
					$author$project$Sharecrop$Types$allVisibilityTags)),
				$author$project$Sharecrop$View$visibilityScopeField(state),
				$author$project$Sharecrop$Ui$label_('Assignee'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$assigneeScopeButton(state.createAssigneeScope),
					$author$project$Sharecrop$View$allAssigneeScopes)),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$author$project$Sharecrop$Ui$testId('create-task')
					]),
				'Create task'),
				A2($author$project$Sharecrop$View$maybeNote, state.createMessage, 'create-message')
			]));
};
var $author$project$Sharecrop$Types$DiscoveryIncludeReservedChanged = function (a) {
	return {$: 'DiscoveryIncludeReservedChanged', a: a};
};
var $author$project$Sharecrop$Types$DiscoveryQueryChanged = function (a) {
	return {$: 'DiscoveryQueryChanged', a: a};
};
var $author$project$Sharecrop$Types$NextDiscoveryPageClicked = {$: 'NextDiscoveryPageClicked'};
var $author$project$Sharecrop$Types$PreviousDiscoveryPageClicked = {$: 'PreviousDiscoveryPageClicked'};
var $author$project$Sharecrop$Types$DiscoveryViewClicked = function (a) {
	return {$: 'DiscoveryViewClicked', a: a};
};
var $author$project$Sharecrop$View$activeAssigneeSuffix = function (item) {
	return (item.activeAssigneeID === '') ? '' : (' · reserved by ' + item.activeAssigneeID);
};
var $author$project$Sharecrop$View$discoveryRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between gap-3 py-2'),
				$author$project$Sharecrop$Ui$testId('discovery-task-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('min-w-0')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('font-medium break-words')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(item.title)
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-500 break-words')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								$author$project$Sharecrop$Labels$taskStateLabel(item.state) + (' · ' + (A3($author$project$Sharecrop$Labels$rewardLabel, item.rewardKind, item.rewardCreditAmount, item.rewardCollectibleCount) + (' · ' + ($author$project$Sharecrop$Labels$participationPolicyLabel(item.participationPolicy) + $author$project$Sharecrop$View$activeAssigneeSuffix(item))))))
							]))
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('shrink-0')
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$DiscoveryViewClicked(item.id)),
								$author$project$Sharecrop$Ui$testId('discovery-view')
							]),
						'View')
					]))
			]));
};
var $author$project$Sharecrop$View$discoveryList = function (tasks) {
	return $elm$core$List$isEmpty(tasks) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('discovery-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('No public tasks available.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('discovery-tasks')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$discoveryRow, tasks));
};
var $elm$core$String$toLower = _String_toLower;
var $author$project$Sharecrop$View$filterTasksByQuery = F2(
	function (query, tasks) {
		var normalized = $elm$core$String$toLower(
			$elm$core$String$trim(query));
		return (normalized === '') ? tasks : A2(
			$elm$core$List$filter,
			function (item) {
				return A2(
					$elm$core$String$contains,
					normalized,
					$elm$core$String$toLower(item.title)) || A2(
					$elm$core$String$contains,
					normalized,
					$elm$core$String$toLower(item.id));
			},
			tasks);
	});
var $author$project$Sharecrop$View$discoveryView = function (state) {
	var visibleTasks = A2($author$project$Sharecrop$View$filterTasksByQuery, state.discoveryQuery, state.discoveryTasks);
	var filtersActive = state.discoveryIncludeReserved || (state.discoveryQuery !== '');
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Discover public tasks'),
				A4(
				$author$project$Sharecrop$Ui$disclosure,
				'discovery-filters',
				filtersActive,
				'Filters',
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$checkbox,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$checked(state.discoveryIncludeReserved),
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$DiscoveryIncludeReservedChanged(!state.discoveryIncludeReserved)),
								$author$project$Sharecrop$Ui$testId('include-reserved')
							]),
						'Include reserved'),
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Search loaded discovery',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('search'),
										$elm$html$Html$Attributes$placeholder('Task title or ID'),
										$elm$html$Html$Attributes$value(state.discoveryQuery),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$DiscoveryQueryChanged),
										$author$project$Sharecrop$Ui$testId('discovery-query')
									]))
							]))
					])),
				A4($author$project$Sharecrop$View$paginationControls, 'discovery-page', $author$project$Sharecrop$Types$PreviousDiscoveryPageClicked, $author$project$Sharecrop$Types$NextDiscoveryPageClicked, state.discoveryOffset),
				$author$project$Sharecrop$View$discoveryList(visibleTasks)
			]));
};
var $author$project$Sharecrop$Types$FundAmountChanged = function (a) {
	return {$: 'FundAmountChanged', a: a};
};
var $author$project$Sharecrop$Types$FundClicked = {$: 'FundClicked'};
var $author$project$Sharecrop$Types$FundOrganizationIdChanged = function (a) {
	return {$: 'FundOrganizationIdChanged', a: a};
};
var $author$project$Sharecrop$Types$FundTaskIdChanged = function (a) {
	return {$: 'FundTaskIdChanged', a: a};
};
var $author$project$Sharecrop$View$fundingView = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$FundClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Fund a task'),
				A4($author$project$Sharecrop$View$taskPicker, 'fund-task-id', state.fundTaskId, $author$project$Sharecrop$Types$FundTaskIdChanged, state.tasks),
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('number'),
						$elm$html$Html$Attributes$placeholder('Amount in credits'),
						$elm$html$Html$Attributes$value(state.fundAmount),
						$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$FundAmountChanged),
						$author$project$Sharecrop$Ui$testId('fund-amount')
					])),
				$author$project$Sharecrop$View$organizationPicker('fund-organization')(state.fundOrganizationId)(state.organizationQuery)($author$project$Sharecrop$Types$FundOrganizationIdChanged)($author$project$Sharecrop$Types$OrganizationQueryChanged)($author$project$Sharecrop$Types$SearchOrganizationsClicked)($author$project$Sharecrop$Types$PreviousOrganizationsPageClicked)($author$project$Sharecrop$Types$NextOrganizationsPageClicked)('Personal balance')(state.organizations)(state.organizationOffset),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$elm$html$Html$Attributes$disabled(state.fundTaskId === ''),
						$author$project$Sharecrop$Ui$testId('fund')
					]),
				'Fund task'),
				A2($author$project$Sharecrop$View$maybeNote, state.fundMessage, 'fund-message')
			]));
};
var $author$project$Sharecrop$Types$NextNotificationsPageClicked = {$: 'NextNotificationsPageClicked'};
var $author$project$Sharecrop$Types$PreviousNotificationsPageClicked = {$: 'PreviousNotificationsPageClicked'};
var $author$project$Sharecrop$Types$MarkNotificationReadClicked = function (a) {
	return {$: 'MarkNotificationReadClicked', a: a};
};
var $author$project$Sharecrop$View$notificationStateClass = function (state) {
	return (state === 'unread') ? 'rounded border border-amber-300 bg-amber-50 px-2 py-1 text-xs font-semibold text-amber-900' : 'rounded border border-slate-200 bg-slate-50 px-2 py-1 text-xs font-semibold text-slate-600';
};
var $author$project$Sharecrop$View$notificationTaskLink = function (notification) {
	var _v0 = A2(
		$elm$json$Json$Decode$decodeString,
		A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string),
		notification.metadataJSON);
	if (_v0.$ === 'Ok') {
		var taskId = _v0.a;
		return A2(
			$elm$html$Html$a,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$href('#/tasks/' + taskId),
					$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
					$author$project$Sharecrop$Ui$testId('notification-task-link')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text('Open task')
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$View$notificationRow = function (notification) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-2 py-3 text-sm'),
				$author$project$Sharecrop$Ui$testId('notification-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-2')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('font-medium text-slate-900')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(notification.kind + (' on ' + notification.subjectKind))
							])),
						A2(
						$elm$html$Html$span,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class(
								$author$project$Sharecrop$View$notificationStateClass(notification.state)),
								$author$project$Sharecrop$Ui$testId('notification-state')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(notification.state)
							]))
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('break-words text-xs text-slate-500')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Subject ' + (notification.subjectID + (' · actor ' + (notification.actorUserID + (' · ' + notification.createdAt)))))
					])),
				$author$project$Sharecrop$View$notificationTaskLink(notification),
				(notification.metadataJSON === '{}') ? $elm$html$Html$text('') : A2(
				$author$project$Sharecrop$Ui$codeBlock,
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$testId('notification-metadata')
					]),
				notification.metadataJSON),
				(notification.state === 'unread') ? A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Sharecrop$Types$MarkNotificationReadClicked(notification.id)),
						$author$project$Sharecrop$Ui$testId('notification-mark-read')
					]),
				'Mark read') : $elm$html$Html$text('')
			]));
};
var $author$project$Sharecrop$View$inboxView = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Inbox'),
				$elm$core$List$isEmpty(state.notifications) ? A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-500'),
						$author$project$Sharecrop$Ui$testId('inbox-empty')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('No notifications.')
					])) : A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
						$author$project$Sharecrop$Ui$testId('inbox-list')
					]),
				A2($elm$core$List$map, $author$project$Sharecrop$View$notificationRow, state.notifications)),
				A4($author$project$Sharecrop$View$paginationControls, 'inbox-page', $author$project$Sharecrop$Types$PreviousNotificationsPageClicked, $author$project$Sharecrop$Types$NextNotificationsPageClicked, state.notificationsOffset),
				A2($author$project$Sharecrop$View$maybeNote, state.inboxMessage, 'inbox-message')
			]));
};
var $author$project$Sharecrop$Types$CreateOrgTeamClicked = {$: 'CreateOrgTeamClicked'};
var $author$project$Sharecrop$Types$CreateOrgTeamNameChanged = function (a) {
	return {$: 'CreateOrgTeamNameChanged', a: a};
};
var $author$project$Sharecrop$Types$ProvisionMemberClicked = {$: 'ProvisionMemberClicked'};
var $author$project$Sharecrop$Types$ProvisionMemberEmailChanged = function (a) {
	return {$: 'ProvisionMemberEmailChanged', a: a};
};
var $author$project$Sharecrop$View$balanceLabel = function (balance) {
	if (balance.$ === 'Just') {
		var amount = balance.a;
		return $elm$core$String$fromInt(amount) + ' credits';
	} else {
		return 'Loading…';
	}
};
var $author$project$Sharecrop$View$collectibleHoldingRow = function (c) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center gap-2 py-2'),
				$author$project$Sharecrop$Ui$testId('collectible-holding-row')
			]),
		_List_fromArray(
			[
				A2($author$project$Sharecrop$Sprites$pixel, c.art, 5),
				A2(
				$elm$html$Html$span,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm font-medium')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(c.name)
					])),
				$author$project$Sharecrop$Ui$badge(
				$author$project$Sharecrop$Labels$collectibleKindLabel(c.kind))
			]));
};
var $author$project$Sharecrop$View$collectiblesHoldingsList = F2(
	function (idPrefix, collectibles) {
		return $elm$core$List$isEmpty(collectibles) ? A2(
			$elm$html$Html$p,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-sm text-slate-500'),
					$author$project$Sharecrop$Ui$testId(idPrefix + '-empty')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text('No collectibles yet.')
				])) : A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
					$author$project$Sharecrop$Ui$testId(idPrefix)
				]),
			A2($elm$core$List$map, $author$project$Sharecrop$View$collectibleHoldingRow, collectibles));
	});
var $author$project$Sharecrop$Types$DeactivateMemberClicked = function (a) {
	return {$: 'DeactivateMemberClicked', a: a};
};
var $author$project$Sharecrop$Types$UpdateMemberRolesClicked = F2(
	function (a, b) {
		return {$: 'UpdateMemberRolesClicked', a: a, b: b};
	});
var $author$project$Sharecrop$View$membershipStatusText = function (status) {
	switch (status.$) {
		case 'MembershipStatusActive':
			return 'active';
		case 'MembershipStatusDeactivated':
			return 'deactivated';
		default:
			return 'removed';
	}
};
var $author$project$Sharecrop$View$organizationRoleText = function (role) {
	switch (role.$) {
		case 'OrganizationRoleOwner':
			return 'owner';
		case 'OrganizationRoleAdmin':
			return 'admin';
		case 'OrganizationRoleMember':
			return 'member';
		case 'OrganizationRoleBilling':
			return 'billing';
		case 'OrganizationRoleReviewer':
			return 'reviewer';
		default:
			return 'public publisher';
	}
};
var $author$project$Sharecrop$View$orgMemberRow = function (member) {
	var roles = $elm$core$List$isEmpty(member.roles) ? 'no roles' : A2(
		$elm$core$String$join,
		', ',
		A2($elm$core$List$map, $author$project$Sharecrop$View$organizationRoleText, member.roles));
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-2 py-2'),
				$author$project$Sharecrop$Ui$testId('org-member-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_Nil,
				_List_fromArray(
					[
						A2(
						$elm$html$Html$a,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$href('#/users/' + member.userID),
								$elm$html$Html$Attributes$class('text-sm font-medium underline'),
								$author$project$Sharecrop$Ui$testId('org-member-link')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(member.userID)
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-600')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								roles + (' · ' + $author$project$Sharecrop$View$membershipStatusText(member.status)))
							]))
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick(
								A2(
									$author$project$Sharecrop$Types$UpdateMemberRolesClicked,
									member.userID,
									_List_fromArray(
										['member']))),
								$author$project$Sharecrop$Ui$testId('member-role-member')
							]),
						'Member'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick(
								A2(
									$author$project$Sharecrop$Types$UpdateMemberRolesClicked,
									member.userID,
									_List_fromArray(
										['member', 'reviewer']))),
								$author$project$Sharecrop$Ui$testId('member-role-reviewer')
							]),
						'Reviewer'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick(
								A2(
									$author$project$Sharecrop$Types$UpdateMemberRolesClicked,
									member.userID,
									_List_fromArray(
										['admin']))),
								$author$project$Sharecrop$Ui$testId('member-role-admin')
							]),
						'Admin'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$DeactivateMemberClicked(member.userID)),
								$author$project$Sharecrop$Ui$testId('deactivate-member')
							]),
						'Deactivate')
					]))
			]));
};
var $author$project$Sharecrop$View$orgMembersList = function (members) {
	return $elm$core$List$isEmpty(members) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('org-members-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('No members yet.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('org-members')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$orgMemberRow, members));
};
var $author$project$Sharecrop$Types$ApplyOrgTaskViewClicked = function (a) {
	return {$: 'ApplyOrgTaskViewClicked', a: a};
};
var $author$project$Sharecrop$Types$NextOrgTasksPageClicked = {$: 'NextOrgTasksPageClicked'};
var $author$project$Sharecrop$Types$OrgTaskQueryChanged = function (a) {
	return {$: 'OrgTaskQueryChanged', a: a};
};
var $author$project$Sharecrop$Types$OrgTaskSavedViewNameChanged = function (a) {
	return {$: 'OrgTaskSavedViewNameChanged', a: a};
};
var $author$project$Sharecrop$Types$OrgTaskSortChanged = function (a) {
	return {$: 'OrgTaskSortChanged', a: a};
};
var $author$project$Sharecrop$Types$OrgTaskTypeFilterChanged = function (a) {
	return {$: 'OrgTaskTypeFilterChanged', a: a};
};
var $author$project$Sharecrop$Types$PreviousOrgTasksPageClicked = {$: 'PreviousOrgTasksPageClicked'};
var $author$project$Sharecrop$Types$SaveOrgTaskViewClicked = {$: 'SaveOrgTaskViewClicked'};
var $author$project$Sharecrop$Types$SearchOrgTasksClicked = {$: 'SearchOrgTasksClicked'};
var $author$project$Sharecrop$Types$OrgTaskFilterChanged = function (a) {
	return {$: 'OrgTaskFilterChanged', a: a};
};
var $author$project$Sharecrop$View$orgTaskFilterButton = F2(
	function (selected, _v0) {
		var tag = _v0.a;
		var labelText = _v0.b;
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, tag),
			$author$project$Sharecrop$Types$OrgTaskFilterChanged(tag),
			'org-task-filter-' + ((tag === '') ? 'all' : tag),
			labelText);
	});
var $author$project$Sharecrop$View$orgTaskFilterOptions = _List_fromArray(
	[
		_Utils_Tuple2('', 'All'),
		_Utils_Tuple2('open', 'Open'),
		_Utils_Tuple2('draft', 'Draft'),
		_Utils_Tuple2('closed', 'Closed')
	]);
var $author$project$Sharecrop$View$queueViewStateLabel = function (value) {
	switch (value) {
		case 'review':
			return 'Review';
		case 'ready':
			return 'Ready';
		case 'assigned':
			return 'Assigned';
		case 'draft':
			return 'Draft';
		case 'open':
			return 'Open';
		case 'closed':
			return 'Closed';
		case 'cancelled':
			return 'Cancelled';
		default:
			return '';
	}
};
var $author$project$Sharecrop$View$queueViewTypeLabel = function (value) {
	return ($elm$core$String$trim(value) === '') ? '' : $author$project$Sharecrop$View$taskTypeLabel(value);
};
var $author$project$Sharecrop$View$sortLabel = function (value) {
	switch (value) {
		case 'newest':
			return 'Newest';
		case 'oldest':
			return 'Oldest';
		case 'title_asc':
			return 'Title A-Z';
		case 'title_desc':
			return 'Title Z-A';
		case 'reward_desc':
			return 'Reward high';
		case 'reward_asc':
			return 'Reward low';
		default:
			return '';
	}
};
var $author$project$Sharecrop$View$queueViewLabel = function (savedView) {
	return A2(
		$elm$core$String$join,
		' · ',
		A2(
			$elm$core$List$cons,
			savedView.name,
			A2(
				$elm$core$List$filter,
				function (part) {
					return $elm$core$String$trim(part) !== '';
				},
				_List_fromArray(
					[
						$author$project$Sharecrop$View$queueViewStateLabel(savedView.stateFilter),
						$author$project$Sharecrop$View$queueViewTypeLabel(savedView.typeFilter),
						$author$project$Sharecrop$View$sortLabel(savedView.sort)
					]))));
};
var $author$project$Sharecrop$View$queueSavedViews = function (config) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-2 rounded-md border border-slate-200 bg-white p-3'),
				$author$project$Sharecrop$Ui$testId(config.prefix + '-saved-views')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit(config.saveClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Saved view',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('text'),
										$elm$html$Html$Attributes$placeholder('View name'),
										$elm$html$Html$Attributes$value(config.nameValue),
										$elm$html$Html$Events$onInput(config.nameChanged),
										$author$project$Sharecrop$Ui$testId(config.prefix + '-saved-view-name')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId(config.prefix + '-save-view')
							]),
						'Save')
					])),
				$elm$core$List$isEmpty(config.views) ? A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-500'),
						$author$project$Sharecrop$Ui$testId(config.prefix + '-saved-views-empty')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('No saved views.')
					])) : A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
						$author$project$Sharecrop$Ui$testId(config.prefix + '-saved-view-list')
					]),
				A2(
					$elm$core$List$map,
					function (savedView) {
						return A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									config.applyClicked(savedView.name)),
									$author$project$Sharecrop$Ui$testId(config.prefix + '-saved-view')
								]),
							$author$project$Sharecrop$View$queueViewLabel(savedView));
					},
					config.views))
			]));
};
var $author$project$Sharecrop$View$taskSortOptions = _List_fromArray(
	[
		_Utils_Tuple2('newest', 'Newest'),
		_Utils_Tuple2('oldest', 'Oldest'),
		_Utils_Tuple2('title_asc', 'Title A-Z'),
		_Utils_Tuple2('title_desc', 'Title Z-A'),
		_Utils_Tuple2('reward_desc', 'Reward high-low'),
		_Utils_Tuple2('reward_asc', 'Reward low-high')
	]);
var $author$project$Sharecrop$View$taskSortSelect = F3(
	function (identifier, selectedSort, change) {
		return A2(
			$author$project$Sharecrop$Ui$fieldLabel,
			'Sort',
			_List_fromArray(
				[
					A2(
					$elm$html$Html$select,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
							$elm$html$Html$Attributes$value(selectedSort),
							$elm$html$Html$Events$onInput(change),
							$author$project$Sharecrop$Ui$testId(identifier)
						]),
					A2(
						$elm$core$List$map,
						$author$project$Sharecrop$View$stringOption(selectedSort),
						$author$project$Sharecrop$View$taskSortOptions))
				]));
	});
var $author$project$Sharecrop$View$taskTypeFilterOptions = _List_fromArray(
	[
		_Utils_Tuple2('', 'All types'),
		_Utils_Tuple2('general', 'General'),
		_Utils_Tuple2('code_review', 'Code review'),
		_Utils_Tuple2('security_review', 'Security review'),
		_Utils_Tuple2('product_review', 'Product review'),
		_Utils_Tuple2('ui_ux_review', 'UI/UX review'),
		_Utils_Tuple2('qa_testing', 'QA testing')
	]);
var $author$project$Sharecrop$View$taskTypeFilterSelect = F3(
	function (identifier, selectedType, change) {
		return A2(
			$author$project$Sharecrop$Ui$fieldLabel,
			'Task type',
			_List_fromArray(
				[
					A2(
					$elm$html$Html$select,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
							$elm$html$Html$Attributes$value(selectedType),
							$elm$html$Html$Events$onInput(change),
							$author$project$Sharecrop$Ui$testId(identifier)
						]),
					A2(
						$elm$core$List$map,
						$author$project$Sharecrop$View$stringOption(selectedType),
						$author$project$Sharecrop$View$taskTypeFilterOptions))
				]));
	});
var $author$project$Sharecrop$View$orgTaskControls = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-2')
			]),
		_List_fromArray(
			[
				A2(
				$author$project$Sharecrop$Ui$fieldLabel,
				'Search organization tasks',
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('search'),
								$elm$html$Html$Attributes$placeholder('Task title or ID'),
								$elm$html$Html$Attributes$value(state.orgTaskQuery),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$OrgTaskQueryChanged),
								$author$project$Sharecrop$Ui$testId('org-task-query')
							]))
					])),
				A3($author$project$Sharecrop$View$taskTypeFilterSelect, 'org-task-type', state.orgTaskTypeFilter, $author$project$Sharecrop$Types$OrgTaskTypeFilterChanged),
				A3($author$project$Sharecrop$View$taskSortSelect, 'org-task-sort', state.orgTaskSort, $author$project$Sharecrop$Types$OrgTaskSortChanged),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$SearchOrgTasksClicked),
								$author$project$Sharecrop$Ui$testId('org-task-search')
							]),
						'Search')
					])),
				A4($author$project$Sharecrop$View$paginationControls, 'org-tasks-page', $author$project$Sharecrop$Types$PreviousOrgTasksPageClicked, $author$project$Sharecrop$Types$NextOrgTasksPageClicked, state.orgTaskOffset),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
						$author$project$Sharecrop$Ui$testId('org-task-filter')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Sharecrop$View$orgTaskFilterButton(state.orgTaskFilter),
					$author$project$Sharecrop$View$orgTaskFilterOptions)),
				$author$project$Sharecrop$View$queueSavedViews(
				{applyClicked: $author$project$Sharecrop$Types$ApplyOrgTaskViewClicked, nameChanged: $author$project$Sharecrop$Types$OrgTaskSavedViewNameChanged, nameValue: state.orgTaskSavedViewName, prefix: 'org-task', saveClicked: $author$project$Sharecrop$Types$SaveOrgTaskViewClicked, views: state.orgTaskSavedViews})
			]));
};
var $author$project$Sharecrop$View$orgTeamsList = function (teams) {
	return $elm$core$List$isEmpty(teams) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('org-teams-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('No teams yet.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('org-teams')
			]),
		A2(
			$elm$core$List$map,
			function (team) {
				return A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/teams/' + team.id),
							$elm$html$Html$Attributes$class('block py-1 text-sm underline'),
							$author$project$Sharecrop$Ui$testId('org-team-row')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(team.name)
						]));
			},
			teams));
};
var $author$project$Sharecrop$View$countMembers = F2(
	function (status, members) {
		return $elm$core$List$length(
			A2(
				$elm$core$List$filter,
				function (member) {
					return _Utils_eq(member.status, status);
				},
				members));
	});
var $author$project$Sharecrop$View$countTasks = F2(
	function (state, tasks) {
		return $elm$core$List$length(
			A2(
				$elm$core$List$filter,
				function (task) {
					return _Utils_eq(task.state, state);
				},
				tasks));
	});
var $author$project$Sharecrop$View$inactiveMemberCount = function (members) {
	return $elm$core$List$length(members) - A2($author$project$Sharecrop$View$countMembers, $author$project$Sharecrop$Generated$Organization$MembershipStatusActive, members);
};
var $author$project$Sharecrop$View$operationMetric = F3(
	function (labelText, valueText, identifier) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('rounded-md bg-slate-50 p-3'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs uppercase text-slate-500')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(labelText)
						])),
					A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-lg font-semibold text-slate-900')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(valueText)
						]))
				]));
	});
var $elm$html$Html$h3 = _VirtualDom_node('h3');
var $author$project$Sharecrop$View$orgAuditEventRow = function (event) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('rounded-md bg-slate-50 p-2 text-sm'),
				$author$project$Sharecrop$Ui$testId('org-audit-event')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('font-medium text-slate-900')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(event.action)
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-500')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(event.subjectKind + (' · ' + event.createdAt))
					]))
			]));
};
var $author$project$Sharecrop$View$orgAuditPanel = function (events) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-2'),
				$author$project$Sharecrop$Ui$testId('org-audit-panel')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$h3,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm font-semibold text-slate-900')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Organization audit')
					])),
				$elm$core$List$isEmpty(events) ? A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-500'),
						$author$project$Sharecrop$Ui$testId('org-audit-empty')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('No audit events.')
					])) : A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2'),
						$author$project$Sharecrop$Ui$testId('org-audit-events')
					]),
				A2($elm$core$List$map, $author$project$Sharecrop$View$orgAuditEventRow, events))
			]));
};
var $author$project$Sharecrop$Types$NextOrgLedgerPageClicked = {$: 'NextOrgLedgerPageClicked'};
var $author$project$Sharecrop$Types$PreviousOrgLedgerPageClicked = {$: 'PreviousOrgLedgerPageClicked'};
var $author$project$Sharecrop$Labels$kindLabel = function (kind) {
	switch (kind.$) {
		case 'LedgerEntryKindSignupGrant':
			return 'Signup grant';
		case 'LedgerEntryKindTaskEscrow':
			return 'Task escrow';
		case 'LedgerEntryKindTaskRefund':
			return 'Task refund';
		case 'LedgerEntryKindTaskPayout':
			return 'Task payout';
		case 'LedgerEntryKindTaskTip':
			return 'Task tip';
		default:
			return 'Manual adjustment';
	}
};
var $elm$html$Html$td = _VirtualDom_node('td');
var $elm$html$Html$tr = _VirtualDom_node('tr');
var $author$project$Sharecrop$View$ledgerRow = function (entry) {
	var amountText = (entry.amount > 0) ? ('+' + $elm$core$String$fromInt(entry.amount)) : $elm$core$String$fromInt(entry.amount);
	var amountClass = (entry.amount < 0) ? 'py-2 text-right tabular-nums text-red-700' : 'py-2 text-right tabular-nums text-green-700';
	return A2(
		$elm$html$Html$tr,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('border-t border-slate-100'),
				$author$project$Sharecrop$Ui$testId('ledger-entry')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$td,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('py-2')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(
						$author$project$Sharecrop$Labels$kindLabel(entry.kind))
					])),
				A2(
				$elm$html$Html$td,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class(amountClass)
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(amountText)
					]))
			]));
};
var $elm$html$Html$table = _VirtualDom_node('table');
var $elm$html$Html$tbody = _VirtualDom_node('tbody');
var $author$project$Sharecrop$View$orgLedgerPanel = F2(
	function (entries, offset) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2'),
					$author$project$Sharecrop$Ui$testId('org-ledger-panel')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$h3,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm font-semibold text-slate-900')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Organization ledger')
						])),
					$elm$core$List$isEmpty(entries) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('org-ledger-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No ledger entries.')
						])) : A2(
					$elm$html$Html$table,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('w-full text-left text-sm')
						]),
					_List_fromArray(
						[
							A2(
							$elm$html$Html$tbody,
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$testId('org-ledger')
								]),
							A2($elm$core$List$map, $author$project$Sharecrop$View$ledgerRow, entries))
						])),
					A4($author$project$Sharecrop$View$paginationControls, 'org-ledger-page', $author$project$Sharecrop$Types$PreviousOrgLedgerPageClicked, $author$project$Sharecrop$Types$NextOrgLedgerPageClicked, offset)
				]));
	});
var $author$project$Sharecrop$View$organizationOperationsDashboard = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-3 rounded-md border border-slate-200 bg-white p-3'),
				$author$project$Sharecrop$Ui$testId('org-operations-dashboard')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Operations'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('grid gap-2 sm:grid-cols-2')
					]),
				_List_fromArray(
					[
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Balance',
						$author$project$Sharecrop$View$balanceLabel(state.orgBalance),
						'org-ops-balance'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Teams',
						$elm$core$String$fromInt(
							$elm$core$List$length(state.orgTeams)),
						'org-ops-teams'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Active members',
						$elm$core$String$fromInt(
							A2($author$project$Sharecrop$View$countMembers, $author$project$Sharecrop$Generated$Organization$MembershipStatusActive, state.orgMembers)),
						'org-ops-members-active'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Inactive members',
						$elm$core$String$fromInt(
							$author$project$Sharecrop$View$inactiveMemberCount(state.orgMembers)),
						'org-ops-members-inactive'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Collectibles',
						$elm$core$String$fromInt(
							$elm$core$List$length(state.orgCollectibles)),
						'org-ops-collectibles'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Draft tasks',
						$elm$core$String$fromInt(
							A2($author$project$Sharecrop$View$countTasks, $author$project$Sharecrop$Generated$Task$TaskStateDraft, state.orgTasks)),
						'org-ops-tasks-draft'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Open tasks',
						$elm$core$String$fromInt(
							A2($author$project$Sharecrop$View$countTasks, $author$project$Sharecrop$Generated$Task$TaskStateOpen, state.orgTasks)),
						'org-ops-tasks-open'),
						A3(
						$author$project$Sharecrop$View$operationMetric,
						'Closed tasks',
						$elm$core$String$fromInt(
							A2($author$project$Sharecrop$View$countTasks, $author$project$Sharecrop$Generated$Task$TaskStateClosed, state.orgTasks)),
						'org-ops-tasks-closed')
					])),
				A2($author$project$Sharecrop$View$orgLedgerPanel, state.orgLedger, state.orgLedgerOffset),
				$author$project$Sharecrop$View$orgAuditPanel(state.orgAuditEvents)
			]));
};
var $author$project$Sharecrop$View$provisionableRoles = _List_fromArray(
	['member', 'reviewer', 'public_publisher', 'billing', 'admin']);
var $author$project$Sharecrop$Types$ToggleProvisionMemberRole = function (a) {
	return {$: 'ToggleProvisionMemberRole', a: a};
};
var $author$project$Sharecrop$View$roleLabel = function (role) {
	if (role === 'public_publisher') {
		return 'public publisher';
	} else {
		return role;
	}
};
var $author$project$Sharecrop$View$roleCheckbox = F2(
	function (selected, role) {
		return A2(
			$author$project$Sharecrop$Ui$checkbox,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$checked(
					A2($elm$core$List$member, role, selected)),
					$elm$html$Html$Events$onCheck(
					function (_v0) {
						return $author$project$Sharecrop$Types$ToggleProvisionMemberRole(role);
					}),
					$author$project$Sharecrop$Ui$testId('provision-role-' + role)
				]),
			$author$project$Sharecrop$View$roleLabel(role));
	});
var $author$project$Sharecrop$View$provisionRolePicker = function (selected) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
			]),
		A2(
			$elm$core$List$map,
			$author$project$Sharecrop$View$roleCheckbox(selected),
			$author$project$Sharecrop$View$provisionableRoles));
};
var $author$project$Sharecrop$View$sectionTitleWithCount = F3(
	function (title, count, identifier) {
		return A2(
			$elm$html$Html$h3,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-lg font-medium'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(
					title + (' (' + ($elm$core$String$fromInt(count) + ')')))
				]));
	});
var $author$project$Sharecrop$View$tasksListSimple = F2(
	function (identifier, tasks) {
		return $elm$core$List$isEmpty(tasks) ? A2(
			$elm$html$Html$p,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-sm text-slate-500'),
					$author$project$Sharecrop$Ui$testId(identifier + '-empty')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text('No tasks yet.')
				])) : A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			A2(
				$elm$core$List$map,
				function (item) {
					return A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('py-1 text-sm'),
								$author$project$Sharecrop$Ui$testId(identifier + '-row')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								item.title + (' · ' + $author$project$Sharecrop$Labels$taskStateLabel(item.state)))
							]));
				},
				tasks));
	});
var $author$project$Sharecrop$View$activeOrganizationView = function (state) {
	return (state.activeOrgId === '') ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 space-y-4 rounded-md bg-slate-50 p-4'),
				$author$project$Sharecrop$Ui$testId('active-organization')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_(
				'Balance: ' + $author$project$Sharecrop$View$balanceLabel(state.orgBalance)),
				$author$project$Sharecrop$View$organizationOperationsDashboard(state),
				A3(
				$author$project$Sharecrop$View$sectionTitleWithCount,
				'Organization tasks',
				$elm$core$List$length(state.orgTasks),
				'org-tasks-heading'),
				$author$project$Sharecrop$View$orgTaskControls(state),
				A2($author$project$Sharecrop$View$tasksListSimple, 'org-tasks', state.orgTasks),
				A2($author$project$Sharecrop$View$maybeNote, state.orgTaskMessage, 'org-task-message'),
				$author$project$Sharecrop$Ui$sectionTitle('Teams'),
				$author$project$Sharecrop$View$orgTeamsList(state.orgTeams),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$CreateOrgTeamClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'New team',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('text'),
										$elm$html$Html$Attributes$placeholder('Team name'),
										$elm$html$Html$Attributes$value(state.createOrgTeamName),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateOrgTeamNameChanged),
										$author$project$Sharecrop$Ui$testId('create-org-team-name')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('create-org-team')
							]),
						'Create team')
					])),
				A2($author$project$Sharecrop$View$maybeNote, state.orgTeamMessage, 'org-team-message'),
				$author$project$Sharecrop$Ui$sectionTitle('Members'),
				$author$project$Sharecrop$View$orgMembersList(state.orgMembers),
				$author$project$Sharecrop$Ui$sectionTitle('Provision a member'),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$ProvisionMemberClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Member email',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('email'),
										$elm$html$Html$Attributes$placeholder('person@example.com'),
										$elm$html$Html$Attributes$value(state.provisionMemberEmail),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$ProvisionMemberEmailChanged),
										$author$project$Sharecrop$Ui$testId('provision-member-email')
									]))
							])),
						$author$project$Sharecrop$View$provisionRolePicker(state.provisionMemberRoles),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('provision-member')
							]),
						'Provision member')
					])),
				A2($author$project$Sharecrop$View$maybeNote, state.provisionMemberMessage, 'provision-member-message'),
				$author$project$Sharecrop$Ui$sectionTitle('Collectibles'),
				A2($author$project$Sharecrop$View$collectiblesHoldingsList, 'org-collectibles', state.orgCollectibles),
				A2($author$project$Sharecrop$View$maybeNote, state.orgCollectiblesMessage, 'org-collectibles-message')
			]));
};
var $author$project$Sharecrop$View$organizationDetailView = function (state) {
	var name = A2(
		$elm$core$Maybe$withDefault,
		state.activeOrgId,
		A2(
			$elm$core$Maybe$map,
			function ($) {
				return $.name;
			},
			$elm$core$List$head(
				A2(
					$elm$core$List$filter,
					function (organization) {
						return _Utils_eq(organization.id, state.activeOrgId);
					},
					state.organizations))));
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/organizations'),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
						$author$project$Sharecrop$Ui$testId('back-organizations')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Back to organizations')
					])),
				$author$project$Sharecrop$Ui$sectionTitle(name),
				$author$project$Sharecrop$View$activeOrganizationView(state)
			]));
};
var $author$project$Sharecrop$Types$CreateOrgClicked = {$: 'CreateOrgClicked'};
var $author$project$Sharecrop$Types$CreateOrgNameChanged = function (a) {
	return {$: 'CreateOrgNameChanged', a: a};
};
var $author$project$Sharecrop$View$organizationRow = function (organization) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between py-2'),
				$author$project$Sharecrop$Ui$testId('organization-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('font-medium')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(organization.name)
					])),
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/organizations/' + organization.id),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
						$author$project$Sharecrop$Ui$testId('select-organization')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Open')
					]))
			]));
};
var $author$project$Sharecrop$View$organizationsList = function (state) {
	return $elm$core$List$isEmpty(state.organizations) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('organizations-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('You do not belong to any organizations yet.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('organizations')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$organizationRow, state.organizations));
};
var $author$project$Sharecrop$View$organizationsView = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Organizations'),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-600')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Organizations you belong to. Create one to own tasks and credits as a team.')
					])),
				$author$project$Sharecrop$View$organizationsList(state),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('mt-3 flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$CreateOrgClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'New organization',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('text'),
										$elm$html$Html$Attributes$placeholder('Organization name'),
										$elm$html$Html$Attributes$value(state.createOrgName),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateOrgNameChanged),
										$author$project$Sharecrop$Ui$testId('create-org-name')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('create-org')
							]),
						'Create organization')
					])),
				A2($author$project$Sharecrop$View$maybeNote, state.orgMessage, 'org-message')
			]));
};
var $author$project$Sharecrop$View$balanceView = function (balance) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_('Balance'),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-3xl font-semibold'),
						$author$project$Sharecrop$Ui$testId('balance')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(
						$author$project$Sharecrop$View$balanceLabel(balance))
					]))
			]));
};
var $author$project$Sharecrop$Types$NextLedgerPageClicked = {$: 'NextLedgerPageClicked'};
var $author$project$Sharecrop$Types$PreviousLedgerPageClicked = {$: 'PreviousLedgerPageClicked'};
var $elm$html$Html$th = _VirtualDom_node('th');
var $elm$html$Html$thead = _VirtualDom_node('thead');
var $author$project$Sharecrop$View$ledgerView = F2(
	function (entries, offset) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Ledger'),
					A2(
					$elm$html$Html$table,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('w-full text-left text-sm')
						]),
					_List_fromArray(
						[
							A2(
							$elm$html$Html$thead,
							_List_Nil,
							_List_fromArray(
								[
									A2(
									$elm$html$Html$tr,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-slate-500')
										]),
									_List_fromArray(
										[
											A2(
											$elm$html$Html$th,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$class('pb-2')
												]),
											_List_fromArray(
												[
													$elm$html$Html$text('Entry')
												])),
											A2(
											$elm$html$Html$th,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$class('pb-2 text-right')
												]),
											_List_fromArray(
												[
													$elm$html$Html$text('Amount')
												]))
										]))
								])),
							A2(
							$elm$html$Html$tbody,
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$testId('ledger')
								]),
							A2($elm$core$List$map, $author$project$Sharecrop$View$ledgerRow, entries))
						])),
					A4($author$project$Sharecrop$View$paginationControls, 'ledger-page', $author$project$Sharecrop$Types$PreviousLedgerPageClicked, $author$project$Sharecrop$Types$NextLedgerPageClicked, offset)
				]));
	});
var $author$project$Sharecrop$View$overviewView = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-6'),
				$author$project$Sharecrop$Ui$testId('overview')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Credit account'),
				$author$project$Sharecrop$View$balanceView(state.balance),
				A2($author$project$Sharecrop$View$ledgerView, state.entries, state.ledgerOffset)
			]));
};
var $author$project$Sharecrop$Types$AddSeriesCommentClicked = function (a) {
	return {$: 'AddSeriesCommentClicked', a: a};
};
var $author$project$Sharecrop$Types$SeriesCommentBodyChanged = function (a) {
	return {$: 'SeriesCommentBodyChanged', a: a};
};
var $author$project$Sharecrop$View$seriesCommentRow = function (comment) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('rounded-md border border-slate-200 bg-white p-3'),
				$author$project$Sharecrop$Ui$testId('series-comment')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/users/' + comment.authorUserID),
						$elm$html$Html$Attributes$class('text-xs font-medium text-slate-600 underline')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(comment.authorUserID)
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-700 break-words')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(comment.body)
					]))
			]));
};
var $author$project$Sharecrop$View$seriesCommentsSection = F3(
	function (seriesId, state, data) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2')
				]),
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Discussion'),
					$elm$core$List$isEmpty(data.comments) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('series-comments-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No comments yet.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$author$project$Sharecrop$Ui$testId('series-comments')
						]),
					A2($elm$core$List$map, $author$project$Sharecrop$View$seriesCommentRow, data.comments)),
					A2(
					$elm$html$Html$form,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$elm$html$Html$Events$onSubmit(
							$author$project$Sharecrop$Types$AddSeriesCommentClicked(seriesId))
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textarea_(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$placeholder('Add a comment'),
									$elm$html$Html$Attributes$value(state.seriesCommentBody),
									$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$SeriesCommentBodyChanged),
									$author$project$Sharecrop$Ui$testId('series-comment-body')
								])),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$author$project$Sharecrop$Ui$testId('add-series-comment')
								]),
							'Comment')
						]))
				]));
	});
var $author$project$Sharecrop$Types$AddSeriesTaskClicked = function (a) {
	return {$: 'AddSeriesTaskClicked', a: a};
};
var $author$project$Sharecrop$Types$AddSeriesTaskIdChanged = function (a) {
	return {$: 'AddSeriesTaskIdChanged', a: a};
};
var $author$project$Sharecrop$Types$SeriesRenameDescriptionChanged = function (a) {
	return {$: 'SeriesRenameDescriptionChanged', a: a};
};
var $author$project$Sharecrop$Types$SeriesRenameTitleChanged = function (a) {
	return {$: 'SeriesRenameTitleChanged', a: a};
};
var $author$project$Sharecrop$Types$UpdateSeriesClicked = function (a) {
	return {$: 'UpdateSeriesClicked', a: a};
};
var $author$project$Sharecrop$Types$CloseSeriesClicked = function (a) {
	return {$: 'CloseSeriesClicked', a: a};
};
var $author$project$Sharecrop$Types$PublishSeriesClicked = function (a) {
	return {$: 'PublishSeriesClicked', a: a};
};
var $author$project$Sharecrop$Types$ReopenSeriesClicked = function (a) {
	return {$: 'ReopenSeriesClicked', a: a};
};
var $author$project$Sharecrop$Types$UnpublishSeriesClicked = function (a) {
	return {$: 'UnpublishSeriesClicked', a: a};
};
var $author$project$Sharecrop$View$seriesStateButtons = function (series) {
	return (series.state === 'draft') ? _List_fromArray(
		[
			A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(
					$author$project$Sharecrop$Types$PublishSeriesClicked(series.id)),
					$author$project$Sharecrop$Ui$testId('series-publish')
				]),
			'Publish')
		]) : ((series.state === 'published') ? _List_fromArray(
		[
			A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(
					$author$project$Sharecrop$Types$UnpublishSeriesClicked(series.id)),
					$author$project$Sharecrop$Ui$testId('series-unpublish')
				]),
			'Unpublish'),
			A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(
					$author$project$Sharecrop$Types$CloseSeriesClicked(series.id)),
					$author$project$Sharecrop$Ui$testId('series-close')
				]),
			'Close')
		]) : ((series.state === 'closed') ? _List_fromArray(
		[
			A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(
					$author$project$Sharecrop$Types$ReopenSeriesClicked(series.id)),
					$author$project$Sharecrop$Ui$testId('series-reopen')
				]),
			'Reopen')
		]) : _List_Nil));
};
var $author$project$Sharecrop$View$seriesCreatorControls = F2(
	function (series, state) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-3 rounded-md bg-slate-50 p-4'),
					$author$project$Sharecrop$Ui$testId('series-creator-controls')
				]),
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Manage series'),
					A2(
					$elm$html$Html$form,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$elm$html$Html$Events$onSubmit(
							$author$project$Sharecrop$Types$UpdateSeriesClicked(series.id))
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Title',
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$textInput(
									_List_fromArray(
										[
											$elm$html$Html$Attributes$type_('text'),
											$elm$html$Html$Attributes$placeholder('Series title'),
											$elm$html$Html$Attributes$value(state.seriesRenameTitle),
											$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$SeriesRenameTitleChanged),
											$author$project$Sharecrop$Ui$testId('series-rename-title')
										]))
								])),
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Description',
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$textarea_(
									_List_fromArray(
										[
											$elm$html$Html$Attributes$placeholder('Description'),
											$elm$html$Html$Attributes$value(state.seriesRenameDescription),
											$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$SeriesRenameDescriptionChanged),
											$author$project$Sharecrop$Ui$testId('series-rename-description')
										]))
								])),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$author$project$Sharecrop$Ui$testId('series-update')
								]),
							'Save changes')
						])),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
						]),
					$author$project$Sharecrop$View$seriesStateButtons(series)),
					A2(
					$elm$html$Html$form,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
							$elm$html$Html$Events$onSubmit(
							$author$project$Sharecrop$Types$AddSeriesTaskClicked(series.id))
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Add task',
							_List_fromArray(
								[
									A4($author$project$Sharecrop$View$taskPicker, 'series-add-task-id', state.addSeriesTaskId, $author$project$Sharecrop$Types$AddSeriesTaskIdChanged, state.tasks)
								])),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$elm$html$Html$Attributes$disabled(state.addSeriesTaskId === ''),
									$author$project$Sharecrop$Ui$testId('series-add-task')
								]),
							'Add task')
						]))
				]));
	});
var $author$project$Sharecrop$Types$MoveSeriesTaskDownClicked = F2(
	function (a, b) {
		return {$: 'MoveSeriesTaskDownClicked', a: a, b: b};
	});
var $author$project$Sharecrop$Types$MoveSeriesTaskUpClicked = F2(
	function (a, b) {
		return {$: 'MoveSeriesTaskUpClicked', a: a, b: b};
	});
var $author$project$Sharecrop$Types$RemoveSeriesTaskClicked = F2(
	function (a, b) {
		return {$: 'RemoveSeriesTaskClicked', a: a, b: b};
	});
var $author$project$Sharecrop$View$seriesTaskRow = F3(
	function (seriesId, isCreator, entry) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-2 py-2'),
					$author$project$Sharecrop$Ui$testId('series-task-row')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/tasks/' + entry.id),
							$elm$html$Html$Attributes$class('w-full text-sm underline break-words sm:w-auto'),
							$author$project$Sharecrop$Ui$testId('series-task-link')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(entry.title + (' · ' + entry.state))
						])),
					isCreator ? A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									A2($author$project$Sharecrop$Types$MoveSeriesTaskUpClicked, seriesId, entry.id)),
									$author$project$Sharecrop$Ui$testId('series-task-up')
								]),
							'Up'),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									A2($author$project$Sharecrop$Types$MoveSeriesTaskDownClicked, seriesId, entry.id)),
									$author$project$Sharecrop$Ui$testId('series-task-down')
								]),
							'Down'),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									A2($author$project$Sharecrop$Types$RemoveSeriesTaskClicked, seriesId, entry.id)),
									$author$project$Sharecrop$Ui$testId('series-remove-task')
								]),
							'Remove')
						])) : $elm$html$Html$text('')
				]));
	});
var $author$project$Sharecrop$View$seriesTasksSection = F3(
	function (seriesId, isCreator, data) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2')
				]),
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Tasks'),
					$elm$core$List$isEmpty(data.tasks) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('series-tasks-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No tasks in this series yet.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
							$author$project$Sharecrop$Ui$testId('series-tasks')
						]),
					A2(
						$elm$core$List$map,
						A2($author$project$Sharecrop$View$seriesTaskRow, seriesId, isCreator),
						data.tasks))
				]));
	});
var $author$project$Sharecrop$View$wrapBadge = F2(
	function (identifier, badge) {
		return A2(
			$elm$html$Html$span,
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[badge]));
	});
var $author$project$Sharecrop$View$seriesDetailView = F2(
	function (seriesId, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/series'),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
							$author$project$Sharecrop$Ui$testId('back-series')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Back to series')
						])),
					function () {
					var _v0 = state.seriesDetail;
					if (_v0.$ === 'Just') {
						var data = _v0.a;
						var isCreator = _Utils_eq(data.series.createdBy, state.subjectId);
						return A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('mt-3 space-y-4'),
									$author$project$Sharecrop$Ui$testId('series-detail')
								]),
							_List_fromArray(
								[
									A2(
									$elm$html$Html$div,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('space-y-2')
										]),
									_List_fromArray(
										[
											A2(
											$elm$html$Html$p,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$class('text-2xl font-semibold'),
													$author$project$Sharecrop$Ui$testId('series-detail-title')
												]),
											_List_fromArray(
												[
													$elm$html$Html$text(data.series.title)
												])),
											A2(
											$author$project$Sharecrop$View$wrapBadge,
											'series-state',
											$author$project$Sharecrop$Ui$badge(data.series.state)),
											A2(
											$elm$html$Html$p,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$class('text-sm text-slate-700')
												]),
											_List_fromArray(
												[
													$elm$html$Html$text(data.series.description)
												]))
										])),
									A3($author$project$Sharecrop$View$seriesTasksSection, seriesId, isCreator, data),
									isCreator ? A2($author$project$Sharecrop$View$seriesCreatorControls, data.series, state) : $elm$html$Html$text(''),
									A3($author$project$Sharecrop$View$seriesCommentsSection, seriesId, state, data),
									A2($author$project$Sharecrop$View$maybeNote, state.seriesMessage, 'series-message')
								]));
					} else {
						var _v1 = state.seriesDetailError;
						if (_v1.$ === 'Just') {
							var message = _v1.a;
							return A2(
								$elm$html$Html$p,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('mt-3 text-sm text-slate-700'),
										$author$project$Sharecrop$Ui$testId('series-detail-missing')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Could not load this series: ' + message)
									]));
						} else {
							return A2(
								$elm$html$Html$p,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('mt-3 text-sm text-slate-500'),
										$author$project$Sharecrop$Ui$testId('series-detail-missing')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Loading series ' + (seriesId + '…'))
									]));
						}
					}
				}()
				]));
	});
var $author$project$Sharecrop$Types$CreateSeriesClicked = {$: 'CreateSeriesClicked'};
var $author$project$Sharecrop$Types$CreateSeriesDescriptionChanged = function (a) {
	return {$: 'CreateSeriesDescriptionChanged', a: a};
};
var $author$project$Sharecrop$Types$CreateSeriesTitleChanged = function (a) {
	return {$: 'CreateSeriesTitleChanged', a: a};
};
var $author$project$Sharecrop$View$seriesRow = function (series) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex flex-wrap items-center justify-between gap-2 py-2'),
				$author$project$Sharecrop$Ui$testId('series-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm font-medium')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(series.title)
							])),
						$author$project$Sharecrop$Ui$badge(series.state)
					])),
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/series/' + series.id),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
						$author$project$Sharecrop$Ui$testId('open-series')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Open')
					]))
			]));
};
var $author$project$Sharecrop$View$seriesListView = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Task series'),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-600')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Group related tasks into an ordered series with its own discussion thread.')
					])),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('mt-3 space-y-2'),
						$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$CreateSeriesClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Title',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('text'),
										$elm$html$Html$Attributes$placeholder('Series title'),
										$elm$html$Html$Attributes$value(state.createSeriesTitle),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateSeriesTitleChanged),
										$author$project$Sharecrop$Ui$testId('series-create-title')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Description',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textarea_(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$placeholder('What is this series about?'),
										$elm$html$Html$Attributes$value(state.createSeriesDescription),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CreateSeriesDescriptionChanged),
										$author$project$Sharecrop$Ui$testId('series-create-description')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('create-series')
							]),
						'Create series'),
						A2($author$project$Sharecrop$View$maybeNote, state.seriesMessage, 'series-message')
					])),
				$author$project$Sharecrop$Ui$sectionTitle('Your series'),
				$elm$core$List$isEmpty(state.seriesList) ? A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-500'),
						$author$project$Sharecrop$Ui$testId('series-empty')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('No series yet.')
					])) : A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
						$author$project$Sharecrop$Ui$testId('series')
					]),
				A2($elm$core$List$map, $author$project$Sharecrop$View$seriesRow, state.seriesList))
			]));
};
var $author$project$Sharecrop$Labels$availabilityKindLabel = function (kind) {
	switch (kind.$) {
		case 'TaskAvailabilityKindAvailable':
			return 'available';
		case 'TaskAvailabilityKindReserved':
			return 'reserved';
		case 'TaskAvailabilityKindAwaitingApproval':
			return 'awaiting approval';
		default:
			return 'closed';
	}
};
var $elm$html$Html$Attributes$rel = _VirtualDom_attribute('rel');
var $elm$html$Html$Attributes$target = $elm$html$Html$Attributes$stringProperty('target');
var $author$project$Sharecrop$View$referenceBlock = function (detail) {
	return (detail.referenceURL === '') ? _List_Nil : _List_fromArray(
		[
			$author$project$Sharecrop$Ui$label_('Reference'),
			A2(
			$elm$html$Html$a,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$href(detail.referenceURL),
					$elm$html$Html$Attributes$target('_blank'),
					$elm$html$Html$Attributes$rel('noopener noreferrer'),
					$elm$html$Html$Attributes$class('text-sm underline break-all'),
					$author$project$Sharecrop$Ui$testId('detail-reference')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(detail.referenceURL)
				]))
		]);
};
var $author$project$Sharecrop$View$seriesLinkBlock = function (detail) {
	return (detail.seriesID === '') ? _List_Nil : _List_fromArray(
		[
			A2(
			$elm$html$Html$a,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$href('#/series/' + detail.seriesID),
					$elm$html$Html$Attributes$class('text-sm underline'),
					$author$project$Sharecrop$Ui$testId('task-series-link')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text('Part of a series')
				]))
		]);
};
var $elm$html$Html$Attributes$download = function (fileName) {
	return A2($elm$html$Html$Attributes$stringProperty, 'download', fileName);
};
var $author$project$Sharecrop$View$attachmentLink = F4(
	function (name, contentType, sizeBytes, dataURL) {
		return A2(
			$elm$html$Html$a,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$href(dataURL),
					$elm$html$Html$Attributes$download(name),
					$elm$html$Html$Attributes$class('rounded border border-slate-200 px-2 py-1 text-xs text-slate-700 underline'),
					$author$project$Sharecrop$Ui$testId('attachment-link')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(
					name + (' · ' + (contentType + (' · ' + ($elm$core$String$fromInt(sizeBytes) + ' bytes')))))
				]));
	});
var $author$project$Sharecrop$View$taskAttachmentLink = function (attachment) {
	return A4($author$project$Sharecrop$View$attachmentLink, attachment.name, attachment.contentType, attachment.sizeBytes, attachment.dataURL);
};
var $author$project$Sharecrop$View$taskAttachmentsBlock = function (detail) {
	return $elm$core$List$isEmpty(detail.attachments) ? _List_Nil : _List_fromArray(
		[
			$author$project$Sharecrop$Ui$label_('Attachments'),
			A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
					$author$project$Sharecrop$Ui$testId('detail-attachments')
				]),
			A2($elm$core$List$map, $author$project$Sharecrop$View$taskAttachmentLink, detail.attachments))
		]);
};
var $author$project$Sharecrop$View$taskInputBlock = function (detail) {
	return ((detail.payloadKind === 'json') && (detail.payloadJson !== '')) ? _List_fromArray(
		[
			$author$project$Sharecrop$Ui$label_('Task input'),
			A2(
			$author$project$Sharecrop$Ui$codeBlock,
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$testId('detail-input')
				]),
			detail.payloadJson)
		]) : _List_Nil;
};
var $author$project$Sharecrop$Types$ToggleTaskIntegration = {$: 'ToggleTaskIntegration'};
var $author$project$Sharecrop$Types$MintTaskTokenClicked = {$: 'MintTaskTokenClicked'};
var $author$project$Sharecrop$Types$CopyClicked = function (a) {
	return {$: 'CopyClicked', a: a};
};
var $author$project$Sharecrop$View$copyButton = function (clipboardText) {
	return A2(
		$author$project$Sharecrop$Ui$secondaryButton,
		_List_fromArray(
			[
				$elm$html$Html$Events$onClick(
				$author$project$Sharecrop$Types$CopyClicked(clipboardText)),
				$author$project$Sharecrop$Ui$testId('copy-command'),
				$elm$html$Html$Attributes$class('w-full sm:w-auto')
			]),
		'Copy');
};
var $author$project$Sharecrop$View$integrationEntry = F3(
	function (description, identifier, command) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-700')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(description)
						])),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId(identifier)
						]),
					command),
					$author$project$Sharecrop$View$copyButton(command)
				]));
	});
var $author$project$Sharecrop$View$mcpSchemaBody = function (taskId) {
	return '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.get_task_schema\",\"arguments\":{\"task_id\":\"' + (taskId + '\"}}}');
};
var $author$project$Sharecrop$View$mcpSubmitBody = function (taskId) {
	return '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.submit_response\",\"arguments\":{\"task_id\":\"' + (taskId + '\",\"response_json\":\"{}\"}}}');
};
var $author$project$Sharecrop$View$restGetCurl = F3(
	function (origin, taskId, token) {
		return 'curl ' + (origin + ('/api/tasks/' + (taskId + (' -H \"Authorization: Bearer ' + (token + '\"')))));
	});
var $author$project$Sharecrop$View$restReserveCurl = F3(
	function (origin, taskId, token) {
		return 'curl -X POST ' + (origin + ('/api/tasks/' + (taskId + ('/reservations -H \"Authorization: Bearer ' + (token + '\"')))));
	});
var $author$project$Sharecrop$View$restSubmitCurl = F3(
	function (origin, taskId, token) {
		return 'curl -X POST ' + (origin + ('/api/tasks/' + (taskId + ('/submissions -H \"Authorization: Bearer ' + (token + '\" -H \"Content-Type: application/json\" -d \'{\"response_json\":\"{}\"}\'')))));
	});
var $author$project$Sharecrop$View$taskIntegrationBody = F3(
	function (origin, taskId, state) {
		var _v0 = state.taskAgentToken;
		if (_v0.$ === 'Nothing') {
			return A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm text-slate-700')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('Create an agent token to get runnable, copy-paste commands for this task.')
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$MintTaskTokenClicked),
								$author$project$Sharecrop$Ui$testId('mint-task-token')
							]),
						'Create agent token')
					]));
		} else {
			var token = _v0.a;
			return A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-4')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('space-y-2')
							]),
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$label_('Agent token'),
								A2(
								$author$project$Sharecrop$Ui$codeBlock,
								_List_fromArray(
									[
										$author$project$Sharecrop$Ui$testId('integration-token')
									]),
								token),
								$author$project$Sharecrop$View$copyButton(token),
								A2(
								$elm$html$Html$p,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('text-sm text-slate-700')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Use this as the Bearer token below. Treat it like a password.')
									])),
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$MintTaskTokenClicked),
										$author$project$Sharecrop$Ui$testId('mint-task-token')
									]),
								'Rotate')
							])),
						$author$project$Sharecrop$Ui$label_('MCP'),
						A3(
						$author$project$Sharecrop$View$integrationEntry,
						'Install the MCP server (add to your .mcp.json or Claude config):',
						'integration-mcp-config',
						A2($author$project$Sharecrop$View$mcpConfig, origin, token)),
						A3(
						$author$project$Sharecrop$View$integrationEntry,
						'Fetch the response schema your submission must match:',
						'integration-mcp-schema',
						$author$project$Sharecrop$View$mcpSchemaBody(taskId)),
						A3(
						$author$project$Sharecrop$View$integrationEntry,
						'Submit your response to this task:',
						'integration-mcp-submit',
						$author$project$Sharecrop$View$mcpSubmitBody(taskId)),
						$author$project$Sharecrop$Ui$label_('REST API'),
						A3(
						$author$project$Sharecrop$View$integrationEntry,
						'Get this task over REST:',
						'integration-rest-get',
						A3($author$project$Sharecrop$View$restGetCurl, origin, taskId, token)),
						A3(
						$author$project$Sharecrop$View$integrationEntry,
						'Reserve this task:',
						'integration-rest-reserve',
						A3($author$project$Sharecrop$View$restReserveCurl, origin, taskId, token)),
						A3(
						$author$project$Sharecrop$View$integrationEntry,
						'Submit your response:',
						'integration-rest-submit',
						A3($author$project$Sharecrop$View$restSubmitCurl, origin, taskId, token))
					]));
		}
	});
var $author$project$Sharecrop$View$taskIntegration = F3(
	function (origin, taskId, state) {
		var indicator = state.taskIntegrationOpen ? ' ▾' : ' ▸';
		var body = state.taskIntegrationOpen ? A3($author$project$Sharecrop$View$taskIntegrationBody, origin, taskId, state) : $elm$html$Html$text('');
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-3'),
					$author$project$Sharecrop$Ui$testId('task-instructions')
				]),
			_List_fromArray(
				[
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$ToggleTaskIntegration),
							$author$project$Sharecrop$Ui$testId('toggle-integration')
						]),
					'API & MCP' + indicator),
					body
				]));
	});
var $author$project$Sharecrop$View$taskTypeBadge = function (detail) {
	return ((detail.taskType === '') || (detail.taskType === 'general')) ? _List_Nil : _List_fromArray(
		[
			A2(
			$elm$html$Html$span,
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$testId('detail-type')
				]),
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$badge(
					$author$project$Sharecrop$View$taskTypeLabel(detail.taskType))
				]))
		]);
};
var $author$project$Sharecrop$View$detailCard = F2(
	function (origin, state) {
		var _v0 = state.detail;
		if (_v0.$ === 'Just') {
			var detail = _v0.a;
			return $author$project$Sharecrop$Ui$card(
				_Utils_ap(
					_List_fromArray(
						[
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-2xl font-semibold'),
									$author$project$Sharecrop$Ui$testId('detail-title')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(detail.title)
								])),
							A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
								]),
							_Utils_ap(
								_List_fromArray(
									[
										$author$project$Sharecrop$Ui$badge(
										$author$project$Sharecrop$Labels$taskStateLabel(detail.state)),
										$author$project$Sharecrop$Ui$badge(
										$author$project$Sharecrop$Labels$availabilityKindLabel(detail.availabilityKind)),
										$author$project$Sharecrop$Ui$badge(
										$author$project$Sharecrop$Labels$participationPolicyLabel(detail.participationPolicy))
									]),
								$author$project$Sharecrop$View$taskTypeBadge(detail))),
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm font-medium')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(
									'Reward: ' + A3($author$project$Sharecrop$Labels$rewardLabel, detail.rewardKind, detail.rewardCreditAmount, detail.rewardCollectibleCount))
								])),
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm text-slate-700')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(detail.description)
								]))
						]),
					_Utils_ap(
						$author$project$Sharecrop$View$referenceBlock(detail),
						_Utils_ap(
							$author$project$Sharecrop$View$seriesLinkBlock(detail),
							_Utils_ap(
								$author$project$Sharecrop$View$taskInputBlock(detail),
								_Utils_ap(
									$author$project$Sharecrop$View$taskAttachmentsBlock(detail),
									_List_fromArray(
										[
											$author$project$Sharecrop$Ui$label_('Response schema'),
											A2(
											$author$project$Sharecrop$Ui$codeBlock,
											_List_fromArray(
												[
													$author$project$Sharecrop$Ui$testId('detail-schema')
												]),
											detail.responseSchemaJson),
											A3($author$project$Sharecrop$View$taskIntegration, origin, detail.id, state)
										])))))));
		} else {
			var _v1 = state.detailError;
			if (_v1.$ === 'Just') {
				var message = _v1.a;
				return $author$project$Sharecrop$Ui$card(
					_List_fromArray(
						[
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm text-slate-700'),
									$author$project$Sharecrop$Ui$testId('detail-error')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text('Could not load this task: ' + message)
								]))
						]));
			} else {
				return $author$project$Sharecrop$Ui$card(
					_List_fromArray(
						[
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm text-slate-500')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text('Loading task…')
								]))
						]));
			}
		}
	});
var $author$project$Sharecrop$Types$ModerationDetailsChanged = function (a) {
	return {$: 'ModerationDetailsChanged', a: a};
};
var $author$project$Sharecrop$Types$ReportTaskClicked = function (a) {
	return {$: 'ReportTaskClicked', a: a};
};
var $author$project$Sharecrop$Types$ModerationReasonChanged = function (a) {
	return {$: 'ModerationReasonChanged', a: a};
};
var $author$project$Sharecrop$View$moderationReasonButton = F2(
	function (selectedReason, _v0) {
		var reason = _v0.a;
		var labelText = _v0.b;
		var selectedClass = _Utils_eq(selectedReason, reason) ? ' ring-2 ring-slate-900' : '';
		return A2(
			$elm$html$Html$button,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(
					$author$project$Sharecrop$Types$ModerationReasonChanged(reason)),
					$elm$html$Html$Attributes$class(
					_Utils_ap($author$project$Sharecrop$Ui$secondaryButtonClass, selectedClass)),
					$author$project$Sharecrop$Ui$testId(
					'moderation-reason-' + $elm$core$String$toLower(labelText))
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Sharecrop$Generated$Moderation$ModerationReasonAbuse = {$: 'ModerationReasonAbuse'};
var $author$project$Sharecrop$Generated$Moderation$ModerationReasonOther = {$: 'ModerationReasonOther'};
var $author$project$Sharecrop$Generated$Moderation$ModerationReasonPII = {$: 'ModerationReasonPII'};
var $author$project$Sharecrop$Generated$Moderation$ModerationReasonSpam = {$: 'ModerationReasonSpam'};
var $author$project$Sharecrop$View$moderationReasonOptions = _List_fromArray(
	[
		_Utils_Tuple2($author$project$Sharecrop$Generated$Moderation$ModerationReasonPolicy, 'Policy'),
		_Utils_Tuple2($author$project$Sharecrop$Generated$Moderation$ModerationReasonSpam, 'Spam'),
		_Utils_Tuple2($author$project$Sharecrop$Generated$Moderation$ModerationReasonAbuse, 'Abuse'),
		_Utils_Tuple2($author$project$Sharecrop$Generated$Moderation$ModerationReasonPII, 'PII'),
		_Utils_Tuple2($author$project$Sharecrop$Generated$Moderation$ModerationReasonOther, 'Other')
	]);
var $author$project$Sharecrop$View$moderationReportCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Just') {
		var detail = _v0.a;
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Report task'),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
							$author$project$Sharecrop$Ui$testId('moderation-reasons')
						]),
					A2(
						$elm$core$List$map,
						$author$project$Sharecrop$View$moderationReasonButton(state.moderationReason),
						$author$project$Sharecrop$View$moderationReasonOptions)),
					$author$project$Sharecrop$Ui$textarea_(
					_List_fromArray(
						[
							$elm$html$Html$Attributes$placeholder('Describe the issue'),
							$elm$html$Html$Attributes$value(state.moderationDetails),
							$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$ModerationDetailsChanged),
							$elm$html$Html$Attributes$rows(4),
							$author$project$Sharecrop$Ui$testId('moderation-details')
						])),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$ReportTaskClicked(detail.id)),
							$author$project$Sharecrop$Ui$testId('report-task')
						]),
					'Submit report'),
					A2($author$project$Sharecrop$View$maybeNote, state.moderationMessage, 'moderation-message')
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$Types$OpenSubmissionComments = function (a) {
	return {$: 'OpenSubmissionComments', a: a};
};
var $author$project$Sharecrop$View$discussionButtonLabel = F2(
	function (state, submissionId) {
		return _Utils_eq(
			state.activeSubmissionCommentsID,
			$elm$core$Maybe$Just(submissionId)) ? 'Discussion open' : 'Discuss';
	});
var $author$project$Sharecrop$View$reviewNoteView = function (note) {
	return $elm$core$String$isEmpty(
		$elm$core$String$trim(note)) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('rounded border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900'),
				$author$project$Sharecrop$Ui$testId('submission-review-note')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(note)
			]));
};
var $author$project$Sharecrop$View$redactedAtSuffix = function (value) {
	return ($elm$core$String$trim(value) === '') ? '' : (' at ' + value);
};
var $author$project$Sharecrop$View$sensitiveFieldView = function (field) {
	return A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-xs text-slate-600'),
				$author$project$Sharecrop$Ui$testId('submission-sensitive-field')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(
				field.path + (' · ' + (field.category + (' · ' + (field.retention + (' · ' + (field.redaction + (' · ' + (field.state + $author$project$Sharecrop$View$redactedAtSuffix(field.redactedAt))))))))))
			]));
};
var $author$project$Sharecrop$View$sensitiveFieldsView = function (fields) {
	return $elm$core$List$isEmpty(fields) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-1 rounded border border-slate-200 bg-slate-50 px-3 py-2'),
				$author$project$Sharecrop$Ui$testId('submission-sensitive-fields')
			]),
		A2(
			$elm$core$List$cons,
			A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs font-semibold text-slate-700')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Sensitive fields')
					])),
			A2($elm$core$List$map, $author$project$Sharecrop$View$sensitiveFieldView, fields)));
};
var $author$project$Sharecrop$View$submissionAttachmentLink = function (attachment) {
	return A4($author$project$Sharecrop$View$attachmentLink, attachment.name, attachment.contentType, attachment.sizeBytes, attachment.dataURL);
};
var $author$project$Sharecrop$View$submissionAttachmentsView = function (attachments) {
	return $elm$core$List$isEmpty(attachments) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
				$author$project$Sharecrop$Ui$testId('submission-attachments')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$submissionAttachmentLink, attachments));
};
var $author$project$Sharecrop$Types$AddSubmissionCommentClicked = function (a) {
	return {$: 'AddSubmissionCommentClicked', a: a};
};
var $author$project$Sharecrop$Types$SubmissionCommentBodyChanged = function (a) {
	return {$: 'SubmissionCommentBodyChanged', a: a};
};
var $author$project$Sharecrop$View$submissionCommentRow = function (comment) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('rounded-md border border-slate-200 bg-white p-3'),
				$author$project$Sharecrop$Ui$testId('submission-comment')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/users/' + comment.authorUserID),
						$elm$html$Html$Attributes$class('text-xs font-medium text-slate-600 underline')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(comment.authorUserID)
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-700 break-words')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(comment.body)
					]))
			]));
};
var $author$project$Sharecrop$View$submissionCommentsThread = F2(
	function (state, submission) {
		return _Utils_eq(
			state.activeSubmissionCommentsID,
			$elm$core$Maybe$Just(submission.id)) ? A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2 rounded-md bg-slate-50 p-3'),
					$author$project$Sharecrop$Ui$testId('submission-comments-thread')
				]),
			_List_fromArray(
				[
					$elm$core$List$isEmpty(state.submissionComments) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('submission-comments-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No comments yet.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$author$project$Sharecrop$Ui$testId('submission-comments')
						]),
					A2($elm$core$List$map, $author$project$Sharecrop$View$submissionCommentRow, state.submissionComments)),
					A2(
					$elm$html$Html$form,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$elm$html$Html$Events$onSubmit(
							$author$project$Sharecrop$Types$AddSubmissionCommentClicked(submission.id))
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textarea_(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$placeholder('Add a comment'),
									$elm$html$Html$Attributes$value(state.submissionCommentBody),
									$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$SubmissionCommentBodyChanged),
									$author$project$Sharecrop$Ui$testId('submission-comment-body')
								])),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$author$project$Sharecrop$Ui$testId('add-submission-comment')
								]),
							'Comment'),
							A2($author$project$Sharecrop$View$maybeNote, state.submissionCommentMessage, 'submission-comment-message')
						]))
				])) : $elm$html$Html$text('');
	});
var $author$project$Sharecrop$View$validationErrorView = function (item) {
	return A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-xs text-red-700')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text(item.path + (': ' + item.message))
			]));
};
var $author$project$Sharecrop$View$validationErrorsView = function (errors) {
	return $elm$core$List$isEmpty(errors) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-1')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$validationErrorView, errors));
};
var $author$project$Sharecrop$View$mySubmissionRow = F2(
	function (state, submission) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2 py-3'),
					$author$project$Sharecrop$Ui$testId('my-submission-row')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex items-center justify-between gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$badge(
							$author$project$Sharecrop$Labels$submissionStateLabel(submission.state)),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									$author$project$Sharecrop$Types$OpenSubmissionComments(submission.id)),
									$author$project$Sharecrop$Ui$testId('my-submission-comments-toggle')
								]),
							A2($author$project$Sharecrop$View$discussionButtonLabel, state, submission.id))
						])),
					$author$project$Sharecrop$View$reviewNoteView(submission.reviewNote),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('my-submission-response')
						]),
					submission.responseJSON),
					$author$project$Sharecrop$View$submissionAttachmentsView(submission.attachments),
					$author$project$Sharecrop$View$validationErrorsView(submission.validationErrors),
					$author$project$Sharecrop$View$sensitiveFieldsView(submission.sensitiveFields),
					A2($author$project$Sharecrop$View$submissionCommentsThread, state, submission)
				]));
	});
var $author$project$Sharecrop$View$mySubmissionsCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Nothing') {
		return $elm$html$Html$text('');
	} else {
		var detail = _v0.a;
		var mine = A2(
			$elm$core$List$filter,
			function (submission) {
				return _Utils_eq(submission.taskID, detail.id);
			},
			state.userSubmissions);
		return $elm$core$List$isEmpty(mine) ? $elm$html$Html$text('') : $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('My submissions'),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
							$author$project$Sharecrop$Ui$testId('my-submissions')
						]),
					A2(
						$elm$core$List$map,
						$author$project$Sharecrop$View$mySubmissionRow(state),
						mine))
				]));
	}
};
var $author$project$Sharecrop$Types$CancelTaskClicked = function (a) {
	return {$: 'CancelTaskClicked', a: a};
};
var $author$project$Sharecrop$Types$OpenTaskClicked = function (a) {
	return {$: 'OpenTaskClicked', a: a};
};
var $author$project$Sharecrop$Types$RefundCollectibleRewardClicked = function (a) {
	return {$: 'RefundCollectibleRewardClicked', a: a};
};
var $author$project$Sharecrop$Types$RefundTaskClicked = function (a) {
	return {$: 'RefundTaskClicked', a: a};
};
var $author$project$Sharecrop$Labels$taskStateGuidance = function (state) {
	switch (state.$) {
		case 'TaskStateDraft':
			return 'Next step: fund this task (if it offers a reward) and then open it so workers can submit.';
		case 'TaskStateOpen':
			return 'Workers can submit now. Review submissions below to accept, request changes, or reject.';
		case 'TaskStateClosed':
			return 'This task is closed. An accepted submission has been settled.';
		case 'TaskStateCancelled':
			return 'This task was cancelled. Any escrowed reward was refunded.';
		default:
			return 'This task expired without an accepted submission.';
	}
};
var $author$project$Sharecrop$View$ownerControlsCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Just') {
		var detail = _v0.a;
		var draftOrOpen = _Utils_eq(detail.state, $author$project$Sharecrop$Generated$Task$TaskStateDraft) || _Utils_eq(detail.state, $author$project$Sharecrop$Generated$Task$TaskStateOpen);
		var refundButton = (draftOrOpen && (detail.rewardKind === 'credit')) ? $elm$core$Maybe$Just(
			A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Sharecrop$Types$RefundTaskClicked(detail.id)),
						$author$project$Sharecrop$Ui$testId('refund-task')
					]),
				'Refund credits')) : ((draftOrOpen && (detail.rewardKind === 'bundle')) ? $elm$core$Maybe$Just(
			A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Sharecrop$Types$RefundTaskClicked(detail.id)),
						$author$project$Sharecrop$Ui$testId('refund-task')
					]),
				'Refund reward')) : ((draftOrOpen && (detail.rewardKind === 'collectible')) ? $elm$core$Maybe$Just(
			A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Sharecrop$Types$RefundCollectibleRewardClicked(detail.id)),
						$author$project$Sharecrop$Ui$testId('refund-collectible')
					]),
				'Refund collectible')) : $elm$core$Maybe$Nothing));
		var buttons = A2(
			$elm$core$List$filterMap,
			$elm$core$Basics$identity,
			_List_fromArray(
				[
					_Utils_eq(detail.state, $author$project$Sharecrop$Generated$Task$TaskStateDraft) ? $elm$core$Maybe$Just(
					A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$OpenTaskClicked(detail.id)),
								$author$project$Sharecrop$Ui$testId('open-task')
							]),
						'Open')) : $elm$core$Maybe$Nothing,
					(_Utils_eq(detail.state, $author$project$Sharecrop$Generated$Task$TaskStateDraft) || (_Utils_eq(detail.state, $author$project$Sharecrop$Generated$Task$TaskStateOpen) && (detail.rewardKind === 'none'))) ? $elm$core$Maybe$Just(
					A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$CancelTaskClicked(detail.id)),
								$author$project$Sharecrop$Ui$testId('cancel-task')
							]),
						'Cancel')) : $elm$core$Maybe$Nothing,
					refundButton
				]));
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Owner controls'),
					A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('rounded-md bg-slate-100 px-3 py-2 text-sm text-slate-700'),
							$author$project$Sharecrop$Ui$testId('task-guidance')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(
							$author$project$Sharecrop$Labels$taskStateGuidance(detail.state))
						])),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
						]),
					buttons),
					A2($author$project$Sharecrop$View$maybeNote, state.taskActionMessage, 'task-action-message')
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$Types$ReserveClicked = function (a) {
	return {$: 'ReserveClicked', a: a};
};
var $author$project$Sharecrop$Types$NextOrgTeamsPageClicked = {$: 'NextOrgTeamsPageClicked'};
var $author$project$Sharecrop$Types$OrgTeamQueryChanged = function (a) {
	return {$: 'OrgTeamQueryChanged', a: a};
};
var $author$project$Sharecrop$Types$PreviousOrgTeamsPageClicked = {$: 'PreviousOrgTeamsPageClicked'};
var $author$project$Sharecrop$Types$ReservationOrganizationIdChanged = function (a) {
	return {$: 'ReservationOrganizationIdChanged', a: a};
};
var $author$project$Sharecrop$Types$ReservationTeamIdChanged = function (a) {
	return {$: 'ReservationTeamIdChanged', a: a};
};
var $author$project$Sharecrop$Types$SearchOrgTeamsClicked = {$: 'SearchOrgTeamsClicked'};
var $author$project$Sharecrop$View$organizationTeamReservationFields = F2(
	function (state, detail) {
		var _v0 = detail.assigneeScope;
		switch (_v0.$) {
			case 'TaskAssigneeScopeOrganizationTeam':
				return A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('grid gap-3 md:grid-cols-2')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Organization',
							_List_fromArray(
								[
									$author$project$Sharecrop$View$organizationPicker('reservation-organization-id')(state.reservationOrganizationId)(state.organizationQuery)($author$project$Sharecrop$Types$ReservationOrganizationIdChanged)($author$project$Sharecrop$Types$OrganizationQueryChanged)($author$project$Sharecrop$Types$SearchOrganizationsClicked)($author$project$Sharecrop$Types$PreviousOrganizationsPageClicked)($author$project$Sharecrop$Types$NextOrganizationsPageClicked)('Choose organization')(state.organizations)(state.organizationOffset)
								])),
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Team',
							_List_fromArray(
								[
									$author$project$Sharecrop$View$teamPicker('reservation-team-id')(state.reservationTeamId)(state.orgTeamQuery)($author$project$Sharecrop$Types$ReservationTeamIdChanged)($author$project$Sharecrop$Types$OrgTeamQueryChanged)($author$project$Sharecrop$Types$SearchOrgTeamsClicked)($author$project$Sharecrop$Types$PreviousOrgTeamsPageClicked)($author$project$Sharecrop$Types$NextOrgTeamsPageClicked)('Choose team')(state.orgTeams)(state.orgTeamOffset)
								]))
						]));
			case 'TaskAssigneeScopeTeam':
				return A2(
					$author$project$Sharecrop$Ui$fieldLabel,
					'Team',
					_List_fromArray(
						[
							$author$project$Sharecrop$View$teamPicker('reservation-team-id')(state.reservationTeamId)(state.standaloneTeamQuery)($author$project$Sharecrop$Types$ReservationTeamIdChanged)($author$project$Sharecrop$Types$StandaloneTeamQueryChanged)($author$project$Sharecrop$Types$SearchStandaloneTeamsClicked)($author$project$Sharecrop$Types$PreviousStandaloneTeamsPageClicked)($author$project$Sharecrop$Types$NextStandaloneTeamsPageClicked)('Choose team')(state.standaloneTeams)(state.standaloneTeamOffset)
						]));
			default:
				return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$View$reservationActionForm = F4(
	function (state, detail, label, id) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-3')
				]),
			_List_fromArray(
				[
					A2($author$project$Sharecrop$View$organizationTeamReservationFields, state, detail),
					A2(
					$author$project$Sharecrop$Ui$primaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$ReserveClicked(detail.id)),
							$author$project$Sharecrop$Ui$testId(id)
						]),
					label)
				]));
	});
var $author$project$Sharecrop$View$reservationAction = F2(
	function (state, detail) {
		var _v0 = detail.viewerAction;
		switch (_v0.$) {
			case 'TaskViewerActionReserve':
				return A4($author$project$Sharecrop$View$reservationActionForm, state, detail, 'Reserve', 'reserve-task');
			case 'TaskViewerActionRequestApproval':
				return A4($author$project$Sharecrop$View$reservationActionForm, state, detail, 'Request approval', 'request-approval');
			default:
				return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$Types$ApproveReservationClicked = function (a) {
	return {$: 'ApproveReservationClicked', a: a};
};
var $author$project$Sharecrop$Types$CancelReservationClicked = function (a) {
	return {$: 'CancelReservationClicked', a: a};
};
var $author$project$Sharecrop$Types$DeclineReservationClicked = function (a) {
	return {$: 'DeclineReservationClicked', a: a};
};
var $author$project$Sharecrop$View$reservationButtons = function (reservation) {
	var _v0 = reservation.state;
	switch (_v0.$) {
		case 'TaskReservationStateRequested':
			return _List_fromArray(
				[
					A2(
					$author$project$Sharecrop$Ui$primaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$ApproveReservationClicked(reservation.id)),
							$author$project$Sharecrop$Ui$testId('approve-reservation')
						]),
					'Approve'),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$DeclineReservationClicked(reservation.id)),
							$author$project$Sharecrop$Ui$testId('decline-reservation')
						]),
					'Decline')
				]);
		case 'TaskReservationStateActive':
			return _List_fromArray(
				[
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$CancelReservationClicked(reservation.id)),
							$author$project$Sharecrop$Ui$testId('cancel-reservation')
						]),
					'Cancel')
				]);
		default:
			return _List_Nil;
	}
};
var $author$project$Sharecrop$View$reservationRow = function (reservation) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between gap-3 py-2'),
				$author$project$Sharecrop$Ui$testId('reservation-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_Nil,
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm font-medium')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								reservation.assigneeID + (' · ' + $author$project$Sharecrop$Labels$assigneeScopeLabel(reservation.assigneeKind)))
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-500')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								$author$project$Sharecrop$Labels$reservationStateLabel(reservation.state))
							]))
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				$author$project$Sharecrop$View$reservationButtons(reservation))
			]));
};
var $author$project$Sharecrop$View$reservationsList = function (reservations) {
	return $elm$core$List$isEmpty(reservations) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('reservations')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$reservationRow, reservations));
};
var $author$project$Sharecrop$Labels$viewerActionLabel = function (action) {
	switch (action.$) {
		case 'TaskViewerActionSubmit':
			return 'submit';
		case 'TaskViewerActionReserve':
			return 'reserve';
		case 'TaskViewerActionRequestApproval':
			return 'request approval';
		case 'TaskViewerActionWait':
			return 'wait';
		default:
			return 'none';
	}
};
var $author$project$Sharecrop$View$reservationCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Just') {
		var detail = _v0.a;
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Reservation'),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$badge(
							$author$project$Sharecrop$Labels$viewerActionLabel(detail.viewerAction)),
							$author$project$Sharecrop$Ui$badge(
							$author$project$Sharecrop$Labels$assigneeScopeLabel(detail.assigneeScope))
						])),
					A2($author$project$Sharecrop$View$reservationAction, state, detail),
					$author$project$Sharecrop$View$reservationsList(state.reservations),
					A2($author$project$Sharecrop$View$maybeNote, state.reservationMessage, 'reservation-message')
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$Types$ReviewBanChanged = function (a) {
	return {$: 'ReviewBanChanged', a: a};
};
var $author$project$Sharecrop$Types$ReviewNoteChanged = function (a) {
	return {$: 'ReviewNoteChanged', a: a};
};
var $author$project$Sharecrop$Types$ReviewPartialCreditChanged = function (a) {
	return {$: 'ReviewPartialCreditChanged', a: a};
};
var $author$project$Sharecrop$Types$ReviewTipChanged = function (a) {
	return {$: 'ReviewTipChanged', a: a};
};
var $author$project$Sharecrop$Types$ReviewTipCollectibleChanged = function (a) {
	return {$: 'ReviewTipCollectibleChanged', a: a};
};
var $author$project$Sharecrop$View$reviewControls = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mb-3 grid gap-3 rounded border border-slate-200 p-3 text-sm')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$label,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('grid gap-1')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$span,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs font-semibold text-slate-600')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('Review note')
							])),
						A2(
						$elm$html$Html$textarea,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('min-h-20 rounded border border-slate-300 px-3 py-2 text-sm'),
								$elm$html$Html$Attributes$rows(3),
								$elm$html$Html$Attributes$value(state.reviewNote),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$ReviewNoteChanged),
								$author$project$Sharecrop$Ui$testId('review-note')
							]),
						_List_Nil)
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('grid gap-2 sm:grid-cols-3')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$label,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('grid gap-1')
							]),
						_List_fromArray(
							[
								A2(
								$elm$html$Html$span,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('text-xs font-semibold text-slate-600')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Partial payout')
									])),
								A2(
								$elm$html$Html$input,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('rounded border border-slate-300 px-3 py-2 text-sm'),
										$elm$html$Html$Attributes$type_('number'),
										$elm$html$Html$Attributes$value(state.reviewPartialCredit),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$ReviewPartialCreditChanged),
										$author$project$Sharecrop$Ui$testId('review-partial-credit')
									]),
								_List_Nil)
							])),
						A2(
						$elm$html$Html$label,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('grid gap-1')
							]),
						_List_fromArray(
							[
								A2(
								$elm$html$Html$span,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('text-xs font-semibold text-slate-600')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Tip')
									])),
								A2(
								$elm$html$Html$input,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('rounded border border-slate-300 px-3 py-2 text-sm'),
										$elm$html$Html$Attributes$type_('number'),
										$elm$html$Html$Attributes$value(state.reviewTip),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$ReviewTipChanged),
										$author$project$Sharecrop$Ui$testId('review-tip')
									]),
								_List_Nil)
							])),
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('pt-6')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$checkbox,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$checked(state.reviewBan),
										$elm$html$Html$Events$onCheck($author$project$Sharecrop$Types$ReviewBanChanged),
										$author$project$Sharecrop$Ui$testId('review-ban')
									]),
								'Ban implementor')
							]))
					])),
				A2(
				$elm$html$Html$label,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('grid gap-1')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$span,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs font-semibold text-slate-600')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('Tip a collectible (optional)')
							])),
						A2(
						$elm$html$Html$select,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$fieldClass),
								$elm$html$Html$Attributes$value(state.reviewTipCollectibleId),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$ReviewTipCollectibleChanged),
								$author$project$Sharecrop$Ui$testId('review-tip-collectible')
							]),
						A2(
							$elm$core$List$cons,
							$author$project$Sharecrop$View$blankOption('No collectible tip'),
							A2(
								$elm$core$List$map,
								function (c) {
									return A2(
										$elm$html$Html$option,
										_List_fromArray(
											[
												$elm$html$Html$Attributes$value(c.id),
												$elm$html$Html$Attributes$selected(
												_Utils_eq(state.reviewTipCollectibleId, c.id))
											]),
										_List_fromArray(
											[
												$elm$html$Html$text(
												c.name + (' · ' + $author$project$Sharecrop$Labels$collectibleKindLabel(c.kind)))
											]));
								},
								state.collectibles)))
					]))
			]));
};
var $author$project$Sharecrop$Types$AcceptClicked = function (a) {
	return {$: 'AcceptClicked', a: a};
};
var $author$project$Sharecrop$Types$RejectClicked = function (a) {
	return {$: 'RejectClicked', a: a};
};
var $author$project$Sharecrop$Types$RequestChangesClicked = function (a) {
	return {$: 'RequestChangesClicked', a: a};
};
var $author$project$Sharecrop$View$reviewButtons = F2(
	function (state, submission) {
		var _v0 = submission.state;
		if (_v0.$ === 'SubmissionStateSubmitted') {
			return A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:justify-end')
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$RequestChangesClicked(submission.id)),
								$elm$html$Html$Attributes$disabled(
								$elm$core$String$trim(state.reviewNote) === ''),
								$author$project$Sharecrop$Ui$testId('request-changes')
							]),
						'Request changes'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$RejectClicked(submission.id)),
								$elm$html$Html$Attributes$disabled(
								$elm$core$String$trim(state.reviewNote) === ''),
								$author$project$Sharecrop$Ui$testId('reject-submission')
							]),
						'Reject'),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Events$onClick(
								$author$project$Sharecrop$Types$AcceptClicked(submission.id)),
								$author$project$Sharecrop$Ui$testId('accept-submission')
							]),
						'Accept')
					]));
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$View$submissionRow = F2(
	function (state, submission) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2 py-3'),
					$author$project$Sharecrop$Ui$testId('submission-row')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex items-center justify-between gap-2')
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$badge(
							$author$project$Sharecrop$Labels$submissionStateLabel(submission.state)),
							A2($author$project$Sharecrop$View$reviewButtons, state, submission)
						])),
					A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-xs text-slate-500')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Submitter: ' + submission.submitterID)
						])),
					$author$project$Sharecrop$View$reviewNoteView(submission.reviewNote),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('submission-response')
						]),
					submission.responseJSON),
					$author$project$Sharecrop$View$submissionAttachmentsView(submission.attachments),
					$author$project$Sharecrop$View$validationErrorsView(submission.validationErrors),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Sharecrop$Types$OpenSubmissionComments(submission.id)),
							$author$project$Sharecrop$Ui$testId('submission-comments-toggle')
						]),
					A2($author$project$Sharecrop$View$discussionButtonLabel, state, submission.id)),
					A2($author$project$Sharecrop$View$submissionCommentsThread, state, submission)
				]));
	});
var $author$project$Sharecrop$View$submissionsList = function (state) {
	return $elm$core$List$isEmpty(state.submissions) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('submissions-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('No submissions to review.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('submissions')
			]),
		A2(
			$elm$core$List$map,
			$author$project$Sharecrop$View$submissionRow(state),
			state.submissions));
};
var $author$project$Sharecrop$View$submissionsCard = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Submissions'),
				$elm$core$List$isEmpty(state.submissions) ? $elm$html$Html$text('') : $author$project$Sharecrop$View$reviewControls(state),
				$author$project$Sharecrop$View$submissionsList(state),
				A2($author$project$Sharecrop$View$maybeNote, state.reviewMessage, 'review-message')
			]));
};
var $author$project$Sharecrop$Types$PickSubmitAttachmentClicked = {$: 'PickSubmitAttachmentClicked'};
var $author$project$Sharecrop$Types$RemoveSubmitAttachmentClicked = function (a) {
	return {$: 'RemoveSubmitAttachmentClicked', a: a};
};
var $author$project$Sharecrop$Types$SubmitClicked = {$: 'SubmitClicked'};
var $author$project$Sharecrop$Types$SubmitInputChanged = function (a) {
	return {$: 'SubmitInputChanged', a: a};
};
var $author$project$Sharecrop$View$submitCardForm = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$SubmitClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Submit a response'),
				$author$project$Sharecrop$Ui$textarea_(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$placeholder('{}'),
						$elm$html$Html$Attributes$value(state.submitInput),
						$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$SubmitInputChanged),
						$elm$html$Html$Attributes$rows(6),
						$author$project$Sharecrop$Ui$testId('detail-submit-input')
					])),
				A5($author$project$Sharecrop$View$selectedAttachmentsView, 'Attachments', state.submitAttachments, $author$project$Sharecrop$Types$PickSubmitAttachmentClicked, $author$project$Sharecrop$Types$RemoveSubmitAttachmentClicked, 'submit-attachments'),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$author$project$Sharecrop$Ui$testId('detail-submit')
					]),
				'Submit response'),
				A2($author$project$Sharecrop$View$maybeNote, state.submitMessage, 'detail-submit-message')
			]));
};
var $author$project$Sharecrop$View$submitCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Nothing') {
		return $elm$html$Html$text('');
	} else {
		var detail = _v0.a;
		return _Utils_eq(detail.state, $author$project$Sharecrop$Generated$Task$TaskStateOpen) ? $author$project$Sharecrop$View$submitCardForm(state) : $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$Types$AddTaskCommentClicked = function (a) {
	return {$: 'AddTaskCommentClicked', a: a};
};
var $author$project$Sharecrop$Types$TaskCommentBodyChanged = function (a) {
	return {$: 'TaskCommentBodyChanged', a: a};
};
var $author$project$Sharecrop$View$taskCommentRow = function (comment) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('rounded-md border border-slate-200 bg-white p-3'),
				$author$project$Sharecrop$Ui$testId('task-comment')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/users/' + comment.authorUserID),
						$elm$html$Html$Attributes$class('text-xs font-medium text-slate-600 underline')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(comment.authorUserID)
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-700 break-words')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(comment.body)
					]))
			]));
};
var $author$project$Sharecrop$View$taskCommentsCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Just') {
		var detail = _v0.a;
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('Discussion'),
					$elm$core$List$isEmpty(state.taskComments) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('task-comments-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No comments yet.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$author$project$Sharecrop$Ui$testId('task-comments')
						]),
					A2($elm$core$List$map, $author$project$Sharecrop$View$taskCommentRow, state.taskComments)),
					A2(
					$elm$html$Html$form,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('space-y-2'),
							$elm$html$Html$Events$onSubmit(
							$author$project$Sharecrop$Types$AddTaskCommentClicked(detail.id))
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textarea_(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$placeholder('Add a comment'),
									$elm$html$Html$Attributes$value(state.taskCommentBody),
									$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$TaskCommentBodyChanged),
									$author$project$Sharecrop$Ui$testId('task-comment-body')
								])),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$author$project$Sharecrop$Ui$testId('add-task-comment')
								]),
							'Comment'),
							A2($author$project$Sharecrop$View$maybeNote, state.taskCommentMessage, 'task-comment-message')
						]))
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Sharecrop$View$taskDetailPageView = F2(
	function (origin, state) {
		var isOwner = A2(
			$elm$core$Maybe$withDefault,
			false,
			A2(
				$elm$core$Maybe$map,
				function (detail) {
					return _Utils_eq(detail.createdBy, state.subjectId);
				},
				state.detail));
		var canReview = A2(
			$elm$core$Maybe$withDefault,
			false,
			A2(
				$elm$core$Maybe$map,
				function (detail) {
					return detail.reviewerAction === 'review';
				},
				state.detail));
		var backHref = isOwner ? '#/tasks' : '#/discovery';
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-6')
				]),
			_Utils_ap(
				_List_fromArray(
					[
						A2(
						$elm$html$Html$a,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$href(backHref),
								$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
								$author$project$Sharecrop$Ui$testId('detail-back')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('Back')
							])),
						A2($author$project$Sharecrop$View$detailCard, origin, state)
					]),
				_Utils_ap(
					isOwner ? _List_fromArray(
						[
							$author$project$Sharecrop$View$ownerControlsCard(state),
							$author$project$Sharecrop$View$submissionsCard(state)
						]) : (canReview ? _List_fromArray(
						[
							$author$project$Sharecrop$View$submissionsCard(state)
						]) : _List_fromArray(
						[
							$author$project$Sharecrop$View$reservationCard(state),
							$author$project$Sharecrop$View$submitCard(state),
							$author$project$Sharecrop$View$mySubmissionsCard(state)
						])),
					_List_fromArray(
						[
							$author$project$Sharecrop$View$taskCommentsCard(state),
							$author$project$Sharecrop$View$moderationReportCard(state)
						]))));
	});
var $author$project$Sharecrop$Types$NextTasksPageClicked = {$: 'NextTasksPageClicked'};
var $author$project$Sharecrop$Types$PreviousTasksPageClicked = {$: 'PreviousTasksPageClicked'};
var $author$project$Sharecrop$Types$TaskListQueryChanged = function (a) {
	return {$: 'TaskListQueryChanged', a: a};
};
var $author$project$Sharecrop$Types$TaskListSortChanged = function (a) {
	return {$: 'TaskListSortChanged', a: a};
};
var $author$project$Sharecrop$Types$TaskListTypeFilterChanged = function (a) {
	return {$: 'TaskListTypeFilterChanged', a: a};
};
var $author$project$Sharecrop$Types$TaskStateFilterChanged = function (a) {
	return {$: 'TaskStateFilterChanged', a: a};
};
var $author$project$Sharecrop$View$taskFilterButton = F2(
	function (selected, _v0) {
		var tag = _v0.a;
		var labelText = _v0.b;
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, tag),
			$author$project$Sharecrop$Types$TaskStateFilterChanged(tag),
			'task-filter-' + ((tag === '') ? 'all' : tag),
			labelText);
	});
var $author$project$Sharecrop$View$taskStateFilterOptions = _List_fromArray(
	[
		_Utils_Tuple2('', 'All'),
		_Utils_Tuple2('open', 'Open'),
		_Utils_Tuple2('draft', 'Draft'),
		_Utils_Tuple2('closed', 'Closed')
	]);
var $author$project$Sharecrop$View$taskRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between gap-3 py-2'),
				$author$project$Sharecrop$Ui$testId('task-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('min-w-0')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('font-medium break-words')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(item.title)
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-500 break-words')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								$author$project$Sharecrop$Labels$taskStateLabel(item.state) + (' · ' + (A3($author$project$Sharecrop$Labels$rewardLabel, item.rewardKind, item.rewardCreditAmount, item.rewardCollectibleCount) + $author$project$Sharecrop$View$activeAssigneeSuffix(item))))
							]))
					])),
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/tasks/' + item.id),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass + ' shrink-0'),
						$author$project$Sharecrop$Ui$testId('view-task')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('View')
					]))
			]));
};
var $author$project$Sharecrop$View$tasksList = function (tasks) {
	return $elm$core$List$isEmpty(tasks) ? A2(
		$elm$html$Html$p,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('text-sm text-slate-500'),
				$author$project$Sharecrop$Ui$testId('tasks-empty')
			]),
		_List_fromArray(
			[
				$elm$html$Html$text('No tasks yet.')
			])) : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('tasks')
			]),
		A2($elm$core$List$map, $author$project$Sharecrop$View$taskRow, tasks));
};
var $author$project$Sharecrop$View$tasksView = F2(
	function (origin, state) {
		var visibleTasks = A2($author$project$Sharecrop$View$filterTasksByQuery, state.taskListQuery, state.tasks);
		var filtersActive = (state.taskStateFilter !== '') || ((state.taskListQuery !== '') || ((state.taskListTypeFilter !== '') || (state.taskListSort !== 'newest')));
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('My tasks'),
					A4(
					$author$project$Sharecrop$Ui$disclosure,
					'tasks-filters',
					filtersActive,
					'Filters',
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$label_('Filter by state'),
							A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
									$author$project$Sharecrop$Ui$testId('task-filter')
								]),
							A2(
								$elm$core$List$map,
								$author$project$Sharecrop$View$taskFilterButton(state.taskStateFilter),
								$author$project$Sharecrop$View$taskStateFilterOptions)),
							A2(
							$author$project$Sharecrop$Ui$fieldLabel,
							'Search loaded tasks',
							_List_fromArray(
								[
									$author$project$Sharecrop$Ui$textInput(
									_List_fromArray(
										[
											$elm$html$Html$Attributes$type_('search'),
											$elm$html$Html$Attributes$placeholder('Task title or ID'),
											$elm$html$Html$Attributes$value(state.taskListQuery),
											$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$TaskListQueryChanged),
											$author$project$Sharecrop$Ui$testId('tasks-query')
										]))
								])),
							A3($author$project$Sharecrop$View$taskTypeFilterSelect, 'tasks-type', state.taskListTypeFilter, $author$project$Sharecrop$Types$TaskListTypeFilterChanged),
							A3($author$project$Sharecrop$View$taskSortSelect, 'tasks-sort', state.taskListSort, $author$project$Sharecrop$Types$TaskListSortChanged)
						])),
					A4($author$project$Sharecrop$View$paginationControls, 'tasks-page', $author$project$Sharecrop$Types$PreviousTasksPageClicked, $author$project$Sharecrop$Types$NextTasksPageClicked, state.taskListOffset),
					$author$project$Sharecrop$View$tasksList(visibleTasks)
				]));
	});
var $author$project$Sharecrop$Types$AddTeamMemberClicked = function (a) {
	return {$: 'AddTeamMemberClicked', a: a};
};
var $author$project$Sharecrop$Types$TeamMemberEmailChanged = function (a) {
	return {$: 'TeamMemberEmailChanged', a: a};
};
var $author$project$Sharecrop$Types$ApplyTeamWorkViewClicked = function (a) {
	return {$: 'ApplyTeamWorkViewClicked', a: a};
};
var $author$project$Sharecrop$Types$NextTeamWorkPageClicked = {$: 'NextTeamWorkPageClicked'};
var $author$project$Sharecrop$Types$PreviousTeamWorkPageClicked = {$: 'PreviousTeamWorkPageClicked'};
var $author$project$Sharecrop$Types$SaveTeamWorkViewClicked = {$: 'SaveTeamWorkViewClicked'};
var $author$project$Sharecrop$Types$SearchTeamWorkClicked = {$: 'SearchTeamWorkClicked'};
var $author$project$Sharecrop$Types$TeamWorkQueryChanged = function (a) {
	return {$: 'TeamWorkQueryChanged', a: a};
};
var $author$project$Sharecrop$Types$TeamWorkSavedViewNameChanged = function (a) {
	return {$: 'TeamWorkSavedViewNameChanged', a: a};
};
var $author$project$Sharecrop$Types$TeamWorkSortChanged = function (a) {
	return {$: 'TeamWorkSortChanged', a: a};
};
var $author$project$Sharecrop$Types$TeamWorkTypeFilterChanged = function (a) {
	return {$: 'TeamWorkTypeFilterChanged', a: a};
};
var $author$project$Sharecrop$View$teamCanActOnTask = function (item) {
	var _v0 = item.viewerAction;
	switch (_v0.$) {
		case 'TaskViewerActionSubmit':
			return true;
		case 'TaskViewerActionReserve':
			return true;
		case 'TaskViewerActionRequestApproval':
			return true;
		default:
			return false;
	}
};
var $author$project$Sharecrop$View$filterTeamWork = F3(
	function (teamId, tag, tasks) {
		switch (tag) {
			case 'review':
				return A2(
					$elm$core$List$filter,
					function (item) {
						return item.reviewerAction !== 'none';
					},
					tasks);
			case 'ready':
				return A2($elm$core$List$filter, $author$project$Sharecrop$View$teamCanActOnTask, tasks);
			case 'assigned':
				return A2(
					$elm$core$List$filter,
					function (item) {
						return _Utils_eq(item.activeAssigneeID, teamId);
					},
					tasks);
			case '':
				return tasks;
			default:
				return _List_Nil;
		}
	});
var $author$project$Sharecrop$Types$TeamWorkFilterChanged = function (a) {
	return {$: 'TeamWorkFilterChanged', a: a};
};
var $author$project$Sharecrop$View$teamWorkFilterButton = F2(
	function (selected, _v0) {
		var tag = _v0.a;
		var labelText = _v0.b;
		return A4(
			$author$project$Sharecrop$View$chooserButton,
			_Utils_eq(selected, tag),
			$author$project$Sharecrop$Types$TeamWorkFilterChanged(tag),
			'team-work-filter-' + ((tag === '') ? 'all' : tag),
			labelText);
	});
var $author$project$Sharecrop$View$teamWorkFilterOptions = _List_fromArray(
	[
		_Utils_Tuple2('', 'All'),
		_Utils_Tuple2('review', 'Review'),
		_Utils_Tuple2('ready', 'Ready'),
		_Utils_Tuple2('assigned', 'Assigned')
	]);
var $author$project$Sharecrop$View$teamWorkSection = F4(
	function (title, identifier, emptyMessage, tasks) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-2'),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			_List_fromArray(
				[
					A3(
					$author$project$Sharecrop$View$sectionTitleWithCount,
					title,
					$elm$core$List$length(tasks),
					identifier + '-heading'),
					$elm$core$List$isEmpty(tasks) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId(identifier + '-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text(emptyMessage)
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('divide-y divide-slate-100')
						]),
					A2($elm$core$List$map, $author$project$Sharecrop$View$taskRow, tasks))
				]));
	});
var $author$project$Sharecrop$View$teamWorkDashboard = F2(
	function (teamId, state) {
		var filteredTasks = A3($author$project$Sharecrop$View$filterTeamWork, teamId, state.teamWorkFilter, state.teamWork);
		var readyForTeam = A2($elm$core$List$filter, $author$project$Sharecrop$View$teamCanActOnTask, filteredTasks);
		var reviewTasks = A2(
			$elm$core$List$filter,
			function (item) {
				return item.reviewerAction !== 'none';
			},
			filteredTasks);
		var assignedToTeam = A2(
			$elm$core$List$filter,
			function (item) {
				return _Utils_eq(item.activeAssigneeID, teamId);
			},
			filteredTasks);
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-4'),
					$author$project$Sharecrop$Ui$testId('team-work-dashboard')
				]),
			_List_fromArray(
				[
					A2(
					$author$project$Sharecrop$Ui$fieldLabel,
					'Search team work',
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textInput(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('search'),
									$elm$html$Html$Attributes$placeholder('Task title or ID'),
									$elm$html$Html$Attributes$value(state.teamWorkQuery),
									$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$TeamWorkQueryChanged),
									$author$project$Sharecrop$Ui$testId('team-work-query')
								]))
						])),
					A3($author$project$Sharecrop$View$taskTypeFilterSelect, 'team-work-type', state.teamWorkTypeFilter, $author$project$Sharecrop$Types$TeamWorkTypeFilterChanged),
					A3($author$project$Sharecrop$View$taskSortSelect, 'team-work-sort', state.teamWorkSort, $author$project$Sharecrop$Types$TeamWorkSortChanged),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$SearchTeamWorkClicked),
									$author$project$Sharecrop$Ui$testId('team-work-search')
								]),
							'Search')
						])),
					A4($author$project$Sharecrop$View$paginationControls, 'team-work-page', $author$project$Sharecrop$Types$PreviousTeamWorkPageClicked, $author$project$Sharecrop$Types$NextTeamWorkPageClicked, state.teamWorkOffset),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex flex-wrap gap-2'),
							$author$project$Sharecrop$Ui$testId('team-work-filter')
						]),
					A2(
						$elm$core$List$map,
						$author$project$Sharecrop$View$teamWorkFilterButton(state.teamWorkFilter),
						$author$project$Sharecrop$View$teamWorkFilterOptions)),
					$author$project$Sharecrop$View$queueSavedViews(
					{applyClicked: $author$project$Sharecrop$Types$ApplyTeamWorkViewClicked, nameChanged: $author$project$Sharecrop$Types$TeamWorkSavedViewNameChanged, nameValue: state.teamWorkSavedViewName, prefix: 'team-work', saveClicked: $author$project$Sharecrop$Types$SaveTeamWorkViewClicked, views: state.teamWorkSavedViews}),
					A4($author$project$Sharecrop$View$teamWorkSection, 'Review queue', 'team-review-queue', 'No submissions waiting for team review.', reviewTasks),
					A4($author$project$Sharecrop$View$teamWorkSection, 'Ready for team', 'team-ready-work', 'No team-visible tasks are ready for action.', readyForTeam),
					A4($author$project$Sharecrop$View$teamWorkSection, 'Assigned to team', 'team-assigned-work', 'No tasks are currently assigned to this team.', assignedToTeam),
					A2($author$project$Sharecrop$View$maybeNote, state.teamWorkMessage, 'team-work-message')
				]));
	});
var $author$project$Sharecrop$View$teamDetailView = F2(
	function (teamId, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					function () {
					var _v0 = state.teamDetail;
					if (_v0.$ === 'Just') {
						var detail = _v0.a;
						return A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('space-y-2'),
									$author$project$Sharecrop$Ui$testId('team-detail')
								]),
							_List_fromArray(
								[
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-2xl font-semibold'),
											$author$project$Sharecrop$Ui$testId('team-detail-name')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text(detail.team.name)
										])),
									$author$project$Sharecrop$Ui$label_('Team ' + detail.team.id),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('Owner kind: ' + detail.team.ownerKind)
										])),
									$author$project$Sharecrop$Ui$sectionTitle('Members'),
									$elm$core$List$isEmpty(detail.members) ? A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm text-slate-500'),
											$author$project$Sharecrop$Ui$testId('team-members-empty')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('No members yet.')
										])) : A2(
									$elm$html$Html$div,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
											$author$project$Sharecrop$Ui$testId('team-members')
										]),
									A2(
										$elm$core$List$map,
										function (memberId) {
											return A2(
												$elm$html$Html$a,
												_List_fromArray(
													[
														$elm$html$Html$Attributes$href('#/users/' + memberId),
														$elm$html$Html$Attributes$class('block py-2 text-sm underline'),
														$author$project$Sharecrop$Ui$testId('team-member-row')
													]),
												_List_fromArray(
													[
														$elm$html$Html$text(memberId)
													]));
										},
										detail.members)),
									((detail.team.ownerKind === 'user') && _Utils_eq(detail.team.ownerUserID, state.subjectId)) ? A2(
									$elm$html$Html$form,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
											$elm$html$Html$Events$onSubmit(
											$author$project$Sharecrop$Types$AddTeamMemberClicked(detail.team.id))
										]),
									_List_fromArray(
										[
											A2(
											$author$project$Sharecrop$Ui$fieldLabel,
											'Add member by email',
											_List_fromArray(
												[
													$author$project$Sharecrop$Ui$textInput(
													_List_fromArray(
														[
															$elm$html$Html$Attributes$type_('email'),
															$elm$html$Html$Attributes$placeholder('person@example.com'),
															$elm$html$Html$Attributes$value(state.teamMemberEmail),
															$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$TeamMemberEmailChanged),
															$author$project$Sharecrop$Ui$testId('team-member-email')
														]))
												])),
											A2(
											$author$project$Sharecrop$Ui$primaryButton,
											_List_fromArray(
												[
													$elm$html$Html$Attributes$type_('submit'),
													$author$project$Sharecrop$Ui$testId('add-team-member')
												]),
											'Add member'),
											A2($author$project$Sharecrop$View$maybeNote, state.teamMemberMessage, 'team-member-message')
										])) : $elm$html$Html$text(''),
									A2($author$project$Sharecrop$View$teamWorkDashboard, detail.team.id, state),
									$author$project$Sharecrop$Ui$sectionTitle('Collectibles'),
									A2($author$project$Sharecrop$View$collectiblesHoldingsList, 'team-collectibles', state.teamCollectibles),
									A2($author$project$Sharecrop$View$maybeNote, state.teamCollectiblesMessage, 'team-collectibles-message')
								]));
					} else {
						var _v1 = state.teamDetailError;
						if (_v1.$ === 'Just') {
							var message = _v1.a;
							return A2(
								$elm$html$Html$p,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('text-sm text-slate-700'),
										$author$project$Sharecrop$Ui$testId('team-detail-missing')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Could not load this team: ' + message)
									]));
						} else {
							return A2(
								$elm$html$Html$p,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$class('text-sm text-slate-500'),
										$author$project$Sharecrop$Ui$testId('team-detail-missing')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text('Loading team ' + (teamId + '…'))
									]));
						}
					}
				}()
				]));
	});
var $author$project$Sharecrop$Types$AccountEmailChanged = function (a) {
	return {$: 'AccountEmailChanged', a: a};
};
var $author$project$Sharecrop$Types$ChangePasswordClicked = {$: 'ChangePasswordClicked'};
var $author$project$Sharecrop$Types$ConfirmEmailVerificationClicked = {$: 'ConfirmEmailVerificationClicked'};
var $author$project$Sharecrop$Types$CurrentPasswordChanged = function (a) {
	return {$: 'CurrentPasswordChanged', a: a};
};
var $author$project$Sharecrop$Types$DeactivateAccountClicked = {$: 'DeactivateAccountClicked'};
var $author$project$Sharecrop$Types$EmailVerificationInputChanged = function (a) {
	return {$: 'EmailVerificationInputChanged', a: a};
};
var $author$project$Sharecrop$Types$NewPasswordChanged = function (a) {
	return {$: 'NewPasswordChanged', a: a};
};
var $author$project$Sharecrop$Types$PrivacyRequestClicked = function (a) {
	return {$: 'PrivacyRequestClicked', a: a};
};
var $author$project$Sharecrop$Generated$Privacy$PrivacyRequestKindDataExport = {$: 'PrivacyRequestKindDataExport'};
var $author$project$Sharecrop$Generated$Privacy$PrivacyRequestKindSensitiveFieldDeletion = {$: 'PrivacyRequestKindSensitiveFieldDeletion'};
var $author$project$Sharecrop$Types$RequestEmailVerificationClicked = {$: 'RequestEmailVerificationClicked'};
var $author$project$Sharecrop$Types$UpdateProfileClicked = {$: 'UpdateProfileClicked'};
var $author$project$Sharecrop$Ui$dangerButtonClass = 'rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50';
var $author$project$Sharecrop$Ui$dangerButton = F2(
	function (attrs, labelText) {
		return A2(
			$elm$html$Html$button,
			A2(
				$elm$core$List$cons,
				$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$dangerButtonClass),
				attrs),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Sharecrop$View$accountSettingsCard = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Account settings'),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2'),
						$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$UpdateProfileClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Email',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('email'),
										$elm$html$Html$Attributes$placeholder('person@example.com'),
										$elm$html$Html$Attributes$value(state.accountEmail),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$AccountEmailChanged),
										$author$project$Sharecrop$Ui$testId('account-email')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('update-profile')
							]),
						'Save profile')
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2')
					]),
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$label_('Email verification'),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$RequestEmailVerificationClicked),
								$author$project$Sharecrop$Ui$testId('request-email-verification')
							]),
						'Create verification token'),
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('text'),
								$elm$html$Html$Attributes$placeholder('Verification token'),
								$elm$html$Html$Attributes$value(state.emailVerificationInput),
								$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$EmailVerificationInputChanged),
								$author$project$Sharecrop$Ui$testId('email-verification-token')
							])),
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('button'),
								$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$ConfirmEmailVerificationClicked),
								$author$project$Sharecrop$Ui$testId('confirm-email-verification')
							]),
						'Verify email')
					])),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2'),
						$elm$html$Html$Events$onSubmit($author$project$Sharecrop$Types$ChangePasswordClicked)
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'Current password',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('password'),
										$elm$html$Html$Attributes$value(state.currentPassword),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$CurrentPasswordChanged),
										$author$project$Sharecrop$Ui$testId('current-password')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$fieldLabel,
						'New password',
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$textInput(
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('password'),
										$elm$html$Html$Attributes$value(state.newPassword),
										$elm$html$Html$Events$onInput($author$project$Sharecrop$Types$NewPasswordChanged),
										$author$project$Sharecrop$Ui$testId('new-password')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('change-password')
							]),
						'Change password')
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('space-y-2')
					]),
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$label_('Privacy requests'),
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
							]),
						_List_fromArray(
							[
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('button'),
										$elm$html$Html$Events$onClick(
										$author$project$Sharecrop$Types$PrivacyRequestClicked($author$project$Sharecrop$Generated$Privacy$PrivacyRequestKindDataExport)),
										$author$project$Sharecrop$Ui$testId('request-data-export')
									]),
								'Request data export'),
								A2(
								$author$project$Sharecrop$Ui$secondaryButton,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$type_('button'),
										$elm$html$Html$Events$onClick(
										$author$project$Sharecrop$Types$PrivacyRequestClicked($author$project$Sharecrop$Generated$Privacy$PrivacyRequestKindSensitiveFieldDeletion)),
										$author$project$Sharecrop$Ui$testId('request-sensitive-deletion')
									]),
								'Request sensitive-field deletion')
							]))
					])),
				A2(
				$author$project$Sharecrop$Ui$dangerButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$DeactivateAccountClicked),
						$author$project$Sharecrop$Ui$testId('deactivate-account')
					]),
				'Deactivate account'),
				A2($author$project$Sharecrop$View$maybeNote, state.accountMessage, 'account-message')
			]));
};
var $author$project$Sharecrop$Types$MintUserTokenClicked = {$: 'MintUserTokenClicked'};
var $author$project$Sharecrop$View$mcpClaudeInstall = F2(
	function (origin, token) {
		return 'claude mcp add --transport http sharecrop ' + (origin + ('/mcp --header \"Authorization: Bearer ' + (token + '\"')));
	});
var $author$project$Sharecrop$View$mcpClaudeUpdate = F2(
	function (origin, token) {
		return 'claude mcp remove sharecrop && ' + A2($author$project$Sharecrop$View$mcpClaudeInstall, origin, token);
	});
var $author$project$Sharecrop$View$userAgentAccessCard = F2(
	function (origin, state) {
		return $author$project$Sharecrop$Ui$card(
			A2(
				$elm$core$List$cons,
				$author$project$Sharecrop$Ui$sectionTitle('Your agent access'),
				A2(
					$elm$core$List$cons,
					A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm text-slate-700')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('A personal agent token lets you drive Sharecrop from an agent (over MCP) or the API. Only you can see it here. Treat it like a password.')
							])),
					function () {
						var _v0 = state.userAgentToken;
						if (_v0.$ === 'Nothing') {
							return _List_fromArray(
								[
									A2(
									$author$project$Sharecrop$Ui$primaryButton,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$type_('button'),
											$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$MintUserTokenClicked),
											$author$project$Sharecrop$Ui$testId('mint-user-token')
										]),
									'Create agent token')
								]);
						} else {
							var token = _v0.a;
							return _List_fromArray(
								[
									$author$project$Sharecrop$Ui$label_('Agent token'),
									A2(
									$author$project$Sharecrop$Ui$codeBlock,
									_List_fromArray(
										[
											$author$project$Sharecrop$Ui$testId('user-token')
										]),
									token),
									$author$project$Sharecrop$View$copyButton(token),
									A2(
									$author$project$Sharecrop$Ui$secondaryButton,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$type_('button'),
											$elm$html$Html$Events$onClick($author$project$Sharecrop$Types$MintUserTokenClicked),
											$author$project$Sharecrop$Ui$testId('mint-user-token')
										]),
									'Rotate token'),
									$author$project$Sharecrop$Ui$label_('Install the MCP'),
									A3(
									$author$project$Sharecrop$View$integrationEntry,
									'Claude Code — add the Sharecrop MCP server:',
									'user-mcp-install',
									A2($author$project$Sharecrop$View$mcpClaudeInstall, origin, token)),
									A3(
									$author$project$Sharecrop$View$integrationEntry,
									'Claude Code — update the server (e.g. after rotating the token):',
									'user-mcp-update',
									A2($author$project$Sharecrop$View$mcpClaudeUpdate, origin, token)),
									A3(
									$author$project$Sharecrop$View$integrationEntry,
									'Or add it to your MCP client config (.mcp.json, Codex, Claude Desktop):',
									'user-mcp-config',
									A2($author$project$Sharecrop$View$mcpConfig, origin, token))
								]);
						}
					}())));
	});
var $author$project$Sharecrop$View$userDetailView = F3(
	function (origin, userId, state) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-6')
				]),
			A2(
				$elm$core$List$cons,
				$author$project$Sharecrop$Ui$card(
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$sectionTitle('User'),
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm font-medium'),
									$author$project$Sharecrop$Ui$testId('user-id')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text(userId)
								])),
							A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
								]),
							_List_fromArray(
								[
									A2(
									$elm$html$Html$a,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$href('#/users/' + (userId + '/work')),
											$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
											$author$project$Sharecrop$Ui$testId('user-work-link')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('Public work')
										])),
									A2(
									$elm$html$Html$a,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$href('#/users/' + (userId + '/submissions')),
											$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
											$author$project$Sharecrop$Ui$testId('user-submissions-link')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('Submissions')
										]))
								])),
							$author$project$Sharecrop$Ui$sectionTitle('Public tasks'),
							function () {
							var _v0 = state.userProfile;
							if (_v0.$ === 'Just') {
								var profile = _v0.a;
								return $elm$core$List$isEmpty(profile.tasks) ? A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm text-slate-500'),
											$author$project$Sharecrop$Ui$testId('user-tasks-empty')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('No public tasks.')
										])) : A2(
									$elm$html$Html$div,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
											$author$project$Sharecrop$Ui$testId('user-tasks')
										]),
									A2(
										$elm$core$List$map,
										function (item) {
											return A2(
												$elm$html$Html$a,
												_List_fromArray(
													[
														$elm$html$Html$Attributes$href('#/tasks/' + item.id),
														$elm$html$Html$Attributes$class('block py-2 text-sm underline'),
														$author$project$Sharecrop$Ui$testId('user-task-row')
													]),
												_List_fromArray(
													[
														$elm$html$Html$text(item.title)
													]));
										},
										profile.tasks));
							} else {
								var _v1 = state.userProfileError;
								if (_v1.$ === 'Just') {
									var message = _v1.a;
									return A2(
										$elm$html$Html$p,
										_List_fromArray(
											[
												$elm$html$Html$Attributes$class('text-sm text-slate-700'),
												$author$project$Sharecrop$Ui$testId('user-profile-error')
											]),
										_List_fromArray(
											[
												$elm$html$Html$text('Could not load this user: ' + message)
											]));
								} else {
									return A2(
										$elm$html$Html$p,
										_List_fromArray(
											[
												$elm$html$Html$Attributes$class('text-sm text-slate-500')
											]),
										_List_fromArray(
											[
												$elm$html$Html$text('Loading…')
											]));
								}
							}
						}()
						])),
				_Utils_eq(userId, state.subjectId) ? _List_fromArray(
					[
						$author$project$Sharecrop$View$accountSettingsCard(state),
						A2($author$project$Sharecrop$View$userAgentAccessCard, origin, state)
					]) : _List_Nil));
	});
var $author$project$Sharecrop$Types$NextUserSubmissionsPageClicked = {$: 'NextUserSubmissionsPageClicked'};
var $author$project$Sharecrop$Types$PreviousUserSubmissionsPageClicked = {$: 'PreviousUserSubmissionsPageClicked'};
var $author$project$Sharecrop$View$isRevisionSubmission = function (submission) {
	return _Utils_eq(submission.state, $author$project$Sharecrop$Generated$Submission$SubmissionStateChangesRequested);
};
var $author$project$Sharecrop$Types$StartRevisionClicked = F2(
	function (a, b) {
		return {$: 'StartRevisionClicked', a: a, b: b};
	});
var $author$project$Sharecrop$View$userSubmissionRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-1 py-2'),
				$author$project$Sharecrop$Ui$testId('user-submission-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('#/tasks/' + item.taskID),
						$elm$html$Html$Attributes$class('text-sm underline'),
						$author$project$Sharecrop$Ui$testId('user-submission-task-link')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Task ' + item.taskID)
					])),
				A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-xs text-slate-600')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(
						$author$project$Sharecrop$Labels$submissionStateLabel(item.state))
					])),
				$author$project$Sharecrop$View$reviewNoteView(item.reviewNote),
				A2(
				$author$project$Sharecrop$Ui$codeBlock,
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$testId('user-submission-response')
					]),
				item.responseJSON),
				$author$project$Sharecrop$View$validationErrorsView(item.validationErrors),
				$author$project$Sharecrop$View$sensitiveFieldsView(item.sensitiveFields)
			]));
};
var $author$project$Sharecrop$View$revisionSubmissionRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-2 py-2'),
				$author$project$Sharecrop$Ui$testId('revision-submission-row')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$View$userSubmissionRow(item),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						A2($author$project$Sharecrop$Types$StartRevisionClicked, item.taskID, item.responseJSON)),
						$author$project$Sharecrop$Ui$testId('revision-resubmit')
					]),
				'Revise')
			]));
};
var $author$project$Sharecrop$View$revisionTimelineRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-1 py-2'),
				$author$project$Sharecrop$Ui$testId('revision-timeline-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
					]),
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$badge(
						$author$project$Sharecrop$Labels$submissionStateLabel(item.state)),
						A2(
						$elm$html$Html$a,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$href('#/tasks/' + item.taskID),
								$elm$html$Html$Attributes$class('text-sm underline'),
								$author$project$Sharecrop$Ui$testId('revision-timeline-task-link')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text('Task ' + item.taskID)
							]))
					])),
				$author$project$Sharecrop$View$reviewNoteView(item.reviewNote),
				$author$project$Sharecrop$View$validationErrorsView(item.validationErrors),
				$author$project$Sharecrop$View$sensitiveFieldsView(item.sensitiveFields)
			]));
};
var $author$project$Sharecrop$View$revisionTimelineView = function (submissions) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-2'),
				$author$project$Sharecrop$Ui$testId('revision-timeline')
			]),
		_List_fromArray(
			[
				A3(
				$author$project$Sharecrop$View$sectionTitleWithCount,
				'Revision timeline',
				$elm$core$List$length(submissions),
				'revision-timeline-heading'),
				$elm$core$List$isEmpty(submissions) ? A2(
				$elm$html$Html$p,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('text-sm text-slate-500'),
						$author$project$Sharecrop$Ui$testId('revision-timeline-empty')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('No submission history.')
					])) : A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('divide-y divide-slate-100')
					]),
				A2($elm$core$List$map, $author$project$Sharecrop$View$revisionTimelineRow, submissions))
			]));
};
var $author$project$Sharecrop$View$userSubmissionsView = F2(
	function (userId, state) {
		var submissions = state.userSubmissions;
		var revisionItems = A2($elm$core$List$filter, $author$project$Sharecrop$View$isRevisionSubmission, submissions);
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/users/' + userId),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
							$author$project$Sharecrop$Ui$testId('back-user')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Back to profile')
						])),
					$author$project$Sharecrop$Ui$sectionTitle('Submissions'),
					A3(
					$author$project$Sharecrop$View$sectionTitleWithCount,
					'Revision inbox',
					$elm$core$List$length(revisionItems),
					'revision-inbox-heading'),
					$elm$core$List$isEmpty(revisionItems) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('revision-inbox-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No requested revisions.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
							$author$project$Sharecrop$Ui$testId('revision-inbox')
						]),
					A2($elm$core$List$map, $author$project$Sharecrop$View$revisionSubmissionRow, revisionItems)),
					$elm$core$List$isEmpty(submissions) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId('user-submissions-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('No submissions.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
							$author$project$Sharecrop$Ui$testId('user-submissions')
						]),
					A2($elm$core$List$map, $author$project$Sharecrop$View$userSubmissionRow, submissions)),
					A4($author$project$Sharecrop$View$paginationControls, 'user-submissions-page', $author$project$Sharecrop$Types$PreviousUserSubmissionsPageClicked, $author$project$Sharecrop$Types$NextUserSubmissionsPageClicked, state.userSubmissionsOffset),
					$author$project$Sharecrop$View$revisionTimelineView(submissions)
				]));
	});
var $author$project$Sharecrop$View$userTaskListView = F4(
	function (heading, identifier, userId, tasks) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('#/users/' + userId),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
							$author$project$Sharecrop$Ui$testId('back-user')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Back to profile')
						])),
					$author$project$Sharecrop$Ui$sectionTitle(heading),
					$elm$core$List$isEmpty(tasks) ? A2(
					$elm$html$Html$p,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('text-sm text-slate-500'),
							$author$project$Sharecrop$Ui$testId(identifier + '-empty')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Nothing to show.')
						])) : A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
							$author$project$Sharecrop$Ui$testId(identifier)
						]),
					A2(
						$elm$core$List$map,
						function (item) {
							return A2(
								$elm$html$Html$a,
								_List_fromArray(
									[
										$elm$html$Html$Attributes$href('#/tasks/' + item.id),
										$elm$html$Html$Attributes$class('block py-2 text-sm underline'),
										$author$project$Sharecrop$Ui$testId(identifier + '-row')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text(
										item.title + (' · ' + $author$project$Sharecrop$Labels$taskStateLabel(item.state)))
									]));
						},
						tasks))
				]));
	});
var $author$project$Sharecrop$View$pageView = F2(
	function (origin, state) {
		var _v0 = state.page;
		switch (_v0.$) {
			case 'OverviewPage':
				return $author$project$Sharecrop$View$overviewView(state);
			case 'TasksPage':
				return A2($author$project$Sharecrop$View$tasksView, origin, state);
			case 'CreateTaskPage':
				return $author$project$Sharecrop$View$createTaskView(state);
			case 'TaskDetailPage':
				return A2($author$project$Sharecrop$View$taskDetailPageView, origin, state);
			case 'DiscoveryPage':
				return $author$project$Sharecrop$View$discoveryView(state);
			case 'FundingPage':
				return $author$project$Sharecrop$View$fundingView(state);
			case 'AgentsPage':
				return A2($author$project$Sharecrop$View$agentsView, origin, state);
			case 'CollectiblesPage':
				return $author$project$Sharecrop$View$collectiblesView(state);
			case 'OrganizationsPage':
				return $author$project$Sharecrop$View$organizationsView(state);
			case 'OrganizationDetailPage':
				return $author$project$Sharecrop$View$organizationDetailView(state);
			case 'UserDetailPage':
				var userId = _v0.a;
				return A3($author$project$Sharecrop$View$userDetailView, origin, userId, state);
			case 'UserWorkPage':
				var userId = _v0.a;
				return A4($author$project$Sharecrop$View$userTaskListView, 'Public work', 'user-work', userId, state.userWork);
			case 'UserSubmissionsPage':
				var userId = _v0.a;
				return A2($author$project$Sharecrop$View$userSubmissionsView, userId, state);
			case 'CollectibleDetailPage':
				var collectibleId = _v0.a;
				return A2($author$project$Sharecrop$View$collectibleDetailView, collectibleId, state);
			case 'SeriesListPage':
				return $author$project$Sharecrop$View$seriesListView(state);
			case 'SeriesDetailPage':
				var seriesId = _v0.a;
				return A2($author$project$Sharecrop$View$seriesDetailView, seriesId, state);
			case 'TeamDetailPage':
				var teamId = _v0.a;
				return A2($author$project$Sharecrop$View$teamDetailView, teamId, state);
			case 'AdminPage':
				return $author$project$Sharecrop$View$adminView(state);
			case 'InboxPage':
				return $author$project$Sharecrop$View$inboxView(state);
			default:
				return $author$project$Sharecrop$Ui$card(
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$sectionTitle('Page not found'),
							A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm text-slate-600')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text('That page does not exist.')
								])),
							A2(
							$elm$html$Html$a,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$href('#/'),
									$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
									$author$project$Sharecrop$Ui$testId('not-found-home')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text('Go to overview')
								]))
						]));
		}
	});
var $author$project$Sharecrop$View$loggedInView = F2(
	function (model, state) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-6')
				]),
			_List_fromArray(
				[
					A4($author$project$Sharecrop$View$navBar, model.demo, state.page, state.subjectId, state.isAdmin),
					A3(
					$elm$html$Html$Keyed$node,
					'div',
					_List_Nil,
					_List_fromArray(
						[
							_Utils_Tuple2(
							$author$project$Sharecrop$Types$pageToPath(state.page),
							A2($author$project$Sharecrop$View$pageView, model.origin, state))
						]))
				]));
	});
var $author$project$Sharecrop$View$sessionView = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedOut') {
		return $author$project$Sharecrop$View$authView(model);
	} else {
		var state = _v0.a;
		return A2($author$project$Sharecrop$View$loggedInView, model, state);
	}
};
var $author$project$Sharecrop$View$view = function (model) {
	return {
		body: _List_fromArray(
			[
				A2(
				$elm$html$Html$main_,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('min-h-screen bg-slate-50 p-4 text-slate-950 sm:p-8')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$div,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('mx-auto max-w-3xl space-y-6')
							]),
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$pageTitle('Sharecrop'),
								$author$project$Sharecrop$View$sessionView(model)
							]))
					]))
			]),
		title: 'Sharecrop'
	};
};
var $author$project$Main$main = $elm$browser$Browser$application(
	{
		init: F3(
			function (flags, url, key) {
				return _Utils_Tuple2(
					A3($author$project$Main$initialModel, flags, key, url),
					$author$project$Sharecrop$Api$postRefresh);
			}),
		onUrlChange: $author$project$Sharecrop$Types$UrlChanged,
		onUrlRequest: $author$project$Sharecrop$Types$LinkClicked,
		subscriptions: function (_v0) {
			return $elm$core$Platform$Sub$none;
		},
		update: $author$project$Main$update,
		view: $author$project$Sharecrop$View$view
	});
_Platform_export({'Main':{'init':$author$project$Main$main(
	A2(
		$elm$json$Json$Decode$andThen,
		function (origin) {
			return A2(
				$elm$json$Json$Decode$andThen,
				function (demo) {
					return $elm$json$Json$Decode$succeed(
						{demo: demo, origin: origin});
				},
				A2($elm$json$Json$Decode$field, 'demo', $elm$json$Json$Decode$bool));
		},
		A2($elm$json$Json$Decode$field, 'origin', $elm$json$Json$Decode$string)))(0)}});}(this));