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
}var $elm$core$Basics$EQ = {$: 'EQ'};
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
var $author$project$Main$LinkClicked = function (a) {
	return {$: 'LinkClicked', a: a};
};
var $author$project$Main$UrlChanged = function (a) {
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
var $elm$json$Json$Decode$field = _Json_decodeField;
var $author$project$Main$LoggedOut = {$: 'LoggedOut'};
var $author$project$Main$AgentsPage = {$: 'AgentsPage'};
var $author$project$Main$CollectibleDetailPage = function (a) {
	return {$: 'CollectibleDetailPage', a: a};
};
var $author$project$Main$CollectiblesPage = {$: 'CollectiblesPage'};
var $author$project$Main$CreateTaskPage = {$: 'CreateTaskPage'};
var $author$project$Main$DiscoveryPage = {$: 'DiscoveryPage'};
var $author$project$Main$FundingPage = {$: 'FundingPage'};
var $author$project$Main$OrganizationDetailPage = function (a) {
	return {$: 'OrganizationDetailPage', a: a};
};
var $author$project$Main$OrganizationsPage = {$: 'OrganizationsPage'};
var $author$project$Main$OverviewPage = {$: 'OverviewPage'};
var $author$project$Main$SeriesDetailPage = function (a) {
	return {$: 'SeriesDetailPage', a: a};
};
var $author$project$Main$TaskDetailPage = function (a) {
	return {$: 'TaskDetailPage', a: a};
};
var $author$project$Main$TasksPage = {$: 'TasksPage'};
var $author$project$Main$TeamDetailPage = function (a) {
	return {$: 'TeamDetailPage', a: a};
};
var $author$project$Main$UserDetailPage = function (a) {
	return {$: 'UserDetailPage', a: a};
};
var $author$project$Main$UserSubmissionsPage = function (a) {
	return {$: 'UserSubmissionsPage', a: a};
};
var $author$project$Main$UserWorkPage = function (a) {
	return {$: 'UserWorkPage', a: a};
};
var $author$project$Main$pageFromUrl = function (url) {
	var _v0 = A2(
		$elm$core$String$split,
		'/',
		A2($elm$core$String$dropLeft, 1, url.path));
	_v0$15:
	while (true) {
		if (_v0.b) {
			if (!_v0.b.b) {
				switch (_v0.a) {
					case 'tasks':
						return $author$project$Main$TasksPage;
					case 'discovery':
						return $author$project$Main$DiscoveryPage;
					case 'funding':
						return $author$project$Main$FundingPage;
					case 'agents':
						return $author$project$Main$AgentsPage;
					case 'collectibles':
						return $author$project$Main$CollectiblesPage;
					case 'organizations':
						return $author$project$Main$OrganizationsPage;
					default:
						break _v0$15;
				}
			} else {
				if (!_v0.b.b.b) {
					switch (_v0.a) {
						case 'tasks':
							if (_v0.b.a === 'new') {
								var _v1 = _v0.b;
								return $author$project$Main$CreateTaskPage;
							} else {
								var _v2 = _v0.b;
								var taskId = _v2.a;
								return $author$project$Main$TaskDetailPage(taskId);
							}
						case 'collectibles':
							var _v3 = _v0.b;
							var collectibleId = _v3.a;
							return $author$project$Main$CollectibleDetailPage(collectibleId);
						case 'series':
							var _v4 = _v0.b;
							var seriesId = _v4.a;
							return $author$project$Main$SeriesDetailPage(seriesId);
						case 'teams':
							var _v5 = _v0.b;
							var teamId = _v5.a;
							return $author$project$Main$TeamDetailPage(teamId);
						case 'organizations':
							var _v6 = _v0.b;
							var organizationId = _v6.a;
							return $author$project$Main$OrganizationDetailPage(organizationId);
						case 'users':
							var _v7 = _v0.b;
							var userId = _v7.a;
							return $author$project$Main$UserDetailPage(userId);
						default:
							break _v0$15;
					}
				} else {
					if ((_v0.a === 'users') && (!_v0.b.b.b.b)) {
						switch (_v0.b.b.a) {
							case 'work':
								var _v8 = _v0.b;
								var userId = _v8.a;
								var _v9 = _v8.b;
								return $author$project$Main$UserWorkPage(userId);
							case 'submissions':
								var _v10 = _v0.b;
								var userId = _v10.a;
								var _v11 = _v10.b;
								return $author$project$Main$UserSubmissionsPage(userId);
							default:
								break _v0$15;
						}
					} else {
						break _v0$15;
					}
				}
			}
		} else {
			break _v0$15;
		}
	}
	return $author$project$Main$OverviewPage;
};
var $author$project$Main$initialModel = F3(
	function (flags, key, url) {
		return {
			authError: $elm$core$Maybe$Nothing,
			email: '',
			key: key,
			origin: flags.origin,
			password: '',
			route: $author$project$Main$pageFromUrl(url),
			session: $author$project$Main$LoggedOut
		};
	});
var $elm$core$Platform$Sub$batch = _Platform_batch;
var $elm$core$Platform$Sub$none = $elm$core$Platform$Sub$batch(_List_Nil);
var $author$project$Main$RefreshReceived = function (a) {
	return {$: 'RefreshReceived', a: a};
};
var $author$project$Sharecrop$Generated$Auth$AuthResponse = F3(
	function (subjectKind, subjectID, accessToken) {
		return {accessToken: accessToken, subjectID: subjectID, subjectKind: subjectKind};
	});
var $elm$json$Json$Decode$map3 = _Json_map3;
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
var $author$project$Sharecrop$Generated$Auth$authResponseDecoder = A4(
	$elm$json$Json$Decode$map3,
	$author$project$Sharecrop$Generated$Auth$AuthResponse,
	A2($elm$json$Json$Decode$field, 'subject_kind', $author$project$Sharecrop$Generated$Auth$subjectKindDecoder),
	A2($elm$json$Json$Decode$field, 'subject_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'access_token', $elm$json$Json$Decode$string));
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
var $author$project$Main$postRefresh = $elm$http$Http$post(
	{
		body: $elm$http$Http$emptyBody,
		expect: A2($elm$http$Http$expectJson, $author$project$Main$RefreshReceived, $author$project$Sharecrop$Generated$Auth$authResponseDecoder),
		url: '/api/auth/refresh'
	});
var $author$project$Main$LoggedIn = function (a) {
	return {$: 'LoggedIn', a: a};
};
var $author$project$Main$OrgMembersReceived = function (a) {
	return {$: 'OrgMembersReceived', a: a};
};
var $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen = {$: 'TaskParticipationPolicyOpen'};
var $elm$core$Platform$Cmd$batch = _Platform_batch;
var $elm$core$Platform$Cmd$none = $elm$core$Platform$Cmd$batch(_List_Nil);
var $author$project$Main$ReviewActionReceived = function (a) {
	return {$: 'ReviewActionReceived', a: a};
};
var $elm$json$Json$Encode$int = _Json_wrap;
var $elm$core$String$trim = _String_trim;
var $elm$core$Maybe$withDefault = F2(
	function (_default, maybe) {
		if (maybe.$ === 'Just') {
			var value = maybe.a;
			return value;
		} else {
			return _default;
		}
	});
var $author$project$Main$intInputOrZero = function (raw) {
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
var $author$project$Main$acceptRequestBody = F3(
	function (submissionId, payoutAmount, tipAmount) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'idempotency_key',
					$elm$json$Json$Encode$string('ui-accept:' + submissionId)),
					_Utils_Tuple2(
					'payout_amount',
					$elm$json$Json$Encode$int(
						$author$project$Main$intInputOrZero(payoutAmount))),
					_Utils_Tuple2(
					'tip_amount',
					$elm$json$Json$Encode$int(
						$author$project$Main$intInputOrZero(tipAmount)))
				]));
	});
var $elm$http$Http$Header = F2(
	function (a, b) {
		return {$: 'Header', a: a, b: b};
	});
var $elm$http$Http$header = $elm$http$Http$Header;
var $author$project$Main$authorizedRequest = F5(
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
var $author$project$Main$postAccept = F5(
	function (token, taskId, submissionId, payoutAmount, tipAmount) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/submissions/' + (submissionId + '/accept'))),
			$elm$http$Http$jsonBody(
				A3($author$project$Main$acceptRequestBody, submissionId, payoutAmount, tipAmount)),
			$elm$http$Http$expectWhatever($author$project$Main$ReviewActionReceived));
	});
var $author$project$Main$updateLoggedIn = F2(
	function (model, change) {
		var _v0 = model.session;
		if (_v0.$ === 'LoggedIn') {
			var state = _v0.a;
			return _Utils_update(
				model,
				{
					session: $author$project$Main$LoggedIn(
						change(state))
				});
		} else {
			return model;
		}
	});
var $author$project$Main$acceptCommand = F3(
	function (model, state, submissionId) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reviewMessage: $elm$core$Maybe$Nothing});
					}),
				A5($author$project$Main$postAccept, state.accessToken, taskId, submissionId, state.reviewPartialCredit, state.reviewTip));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Main$AwardReceived = function (a) {
	return {$: 'AwardReceived', a: a};
};
var $author$project$Sharecrop$Generated$Collectible$CollectibleResponse = F6(
	function (id, name, kind, state, transferPolicy, ownerID) {
		return {id: id, kind: kind, name: name, ownerID: ownerID, state: state, transferPolicy: transferPolicy};
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
var $elm$json$Json$Decode$map6 = _Json_map6;
var $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder = A7(
	$elm$json$Json$Decode$map6,
	$author$project$Sharecrop$Generated$Collectible$CollectibleResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'name', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'kind', $author$project$Sharecrop$Generated$Collectible$collectibleKindDecoder),
	A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Collectible$collectibleStateDecoder),
	A2($elm$json$Json$Decode$field, 'transfer_policy', $author$project$Sharecrop$Generated$Collectible$collectibleTransferPolicyDecoder),
	A2($elm$json$Json$Decode$field, 'owner_id', $elm$json$Json$Decode$string));
var $author$project$Main$collectibleRewardRequestBody = function (collectibleId) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'collectible_id',
				$elm$json$Json$Encode$string(collectibleId))
			]));
};
var $author$project$Main$postCollectibleReward = F3(
	function (token, taskId, collectibleId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/collectible-reward'),
			$elm$http$Http$jsonBody(
				$author$project$Main$collectibleRewardRequestBody(collectibleId)),
			A2($elm$http$Http$expectJson, $author$project$Main$AwardReceived, $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder));
	});
var $author$project$Main$awardCommand = F3(
	function (model, state, collectibleId) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.awardTaskId)) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
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
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{awardMessage: $elm$core$Maybe$Nothing});
				}),
			A3($author$project$Main$postCollectibleReward, state.accessToken, state.awardTaskId, collectibleId));
	});
var $author$project$Main$collectibleStateLabel = function (state) {
	switch (state.$) {
		case 'CollectibleStateMinted':
			return 'minted';
		case 'CollectibleStateEscrowed':
			return 'escrowed';
		default:
			return 'awarded';
	}
};
var $author$project$Main$awardSuccessLabel = function (collectible) {
	return 'Awarded ' + (collectible.name + (' (' + ($author$project$Main$collectibleStateLabel(collectible.state) + ').')));
};
var $author$project$Main$balanceFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return $elm$core$Maybe$Just(response.amount);
	} else {
		return $elm$core$Maybe$Nothing;
	}
};
var $author$project$Main$collectiblesFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.collectibles;
	} else {
		return _List_Nil;
	}
};
var $elm$core$List$isEmpty = function (xs) {
	if (!xs.b) {
		return true;
	} else {
		return false;
	}
};
var $author$project$Main$AgentCreated = function (a) {
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
var $elm$json$Json$Decode$list = _Json_decodeList;
var $elm$json$Json$Decode$map4 = _Json_map4;
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
var $elm$json$Json$Encode$list = F2(
	function (func, entries) {
		return _Json_wrap(
			A3(
				$elm$core$List$foldl,
				_Json_addEntry(func),
				_Json_emptyArray(_Utils_Tuple0),
				entries));
	});
var $author$project$Main$agentRequestBody = F2(
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
var $author$project$Main$postAgent = F3(
	function (token, agentLabel, scopes) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/agent-credentials',
			$elm$http$Http$jsonBody(
				A2($author$project$Main$agentRequestBody, agentLabel, scopes)),
			A2($elm$http$Http$expectJson, $author$project$Main$AgentCreated, $author$project$Sharecrop$Generated$Agent$agentCredentialCreatedResponseDecoder));
	});
var $author$project$Main$createAgentCommand = F2(
	function (model, state) {
		return $elm$core$List$isEmpty(state.agentScopes) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
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
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{agentMessage: $elm$core$Maybe$Nothing, newCredential: $elm$core$Maybe$Nothing});
				}),
			A3($author$project$Main$postAgent, state.accessToken, state.agentLabel, state.agentScopes));
	});
var $author$project$Main$CreateOrgReceived = function (a) {
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
var $author$project$Main$createOrgCommand = F2(
	function (model, state) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.createOrgName)) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
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
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{orgMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Main$authorizedRequest,
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
				A2($elm$http$Http$expectJson, $author$project$Main$CreateOrgReceived, $author$project$Sharecrop$Generated$Organization$organizationResponseDecoder)));
	});
var $author$project$Main$CreateOrgTeamReceived = function (a) {
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
var $author$project$Main$createOrgTeamCommand = F2(
	function (model, state) {
		return ($elm$core$String$isEmpty(
			$elm$core$String$trim(state.createOrgTeamName)) || (state.activeOrgId === '')) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
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
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{orgTeamMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Main$authorizedRequest,
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
				A2($elm$http$Http$expectJson, $author$project$Main$CreateOrgTeamReceived, $author$project$Sharecrop$Generated$Team$teamResponseDecoder)));
	});
var $author$project$Main$CreateTaskReceived = function (a) {
	return {$: 'CreateTaskReceived', a: a};
};
var $author$project$Main$createOwnerBody = function (state) {
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
var $author$project$Main$assigneeScopeTag = function (scope) {
	if (scope.$ === 'TaskAssigneeScopeUser') {
		return 'user';
	} else {
		return 'organization_team';
	}
};
var $author$project$Main$reservationHoursValue = function (raw) {
	var _v0 = $elm$core$String$toInt(raw);
	if (_v0.$ === 'Just') {
		var hours = _v0.a;
		return hours;
	} else {
		return 48;
	}
};
var $author$project$Main$createParticipationBody = function (state) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'policy',
				$elm$json$Json$Encode$string(state.createParticipationPolicy)),
				_Utils_Tuple2(
				'assignee_scope',
				$elm$json$Json$Encode$string(
					$author$project$Main$assigneeScopeTag(state.createAssigneeScope))),
				_Utils_Tuple2(
				'reservation_expiry_hours',
				$elm$json$Json$Encode$int(
					$author$project$Main$reservationHoursValue(state.createReservationHours)))
			]));
};
var $author$project$Main$createRewardBody = function (rawAmount) {
	var _v0 = $elm$core$String$toInt(rawAmount);
	if (_v0.$ === 'Just') {
		var amount = _v0.a;
		return (amount > 0) ? $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'kind',
					$elm$json$Json$Encode$string('credit')),
					_Utils_Tuple2(
					'credit_amount',
					$elm$json$Json$Encode$int(amount))
				])) : $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'kind',
					$elm$json$Json$Encode$string('none')),
					_Utils_Tuple2(
					'credit_amount',
					$elm$json$Json$Encode$int(0))
				]));
	} else {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'kind',
					$elm$json$Json$Encode$string('none')),
					_Utils_Tuple2(
					'credit_amount',
					$elm$json$Json$Encode$int(0))
				]));
	}
};
var $author$project$Main$visibilityOrganizationTag = 'organization';
var $author$project$Main$visibilityTeamTag = 'team';
var $author$project$Main$visibilityUserTag = 'user';
var $author$project$Main$createVisibilityBody = function (state) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'kind',
				$elm$json$Json$Encode$string(state.createVisibility)),
				_Utils_Tuple2(
				'user_id',
				$elm$json$Json$Encode$string(
					_Utils_eq(state.createVisibility, $author$project$Main$visibilityUserTag) ? state.createScopeUserId : '')),
				_Utils_Tuple2(
				'team_id',
				$elm$json$Json$Encode$string(
					_Utils_eq(state.createVisibility, $author$project$Main$visibilityTeamTag) ? state.createScopeTeamId : '')),
				_Utils_Tuple2(
				'organization_id',
				$elm$json$Json$Encode$string(
					_Utils_eq(state.createVisibility, $author$project$Main$visibilityOrganizationTag) ? state.createScopeOrganizationId : ''))
			]));
};
var $author$project$Main$createTaskRequestBody = function (state) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'owner',
				$author$project$Main$createOwnerBody(state)),
				_Utils_Tuple2(
				'title',
				$elm$json$Json$Encode$string(state.createTitle)),
				_Utils_Tuple2(
				'description',
				$elm$json$Json$Encode$string(state.createDescription)),
				_Utils_Tuple2(
				'reward',
				$author$project$Main$createRewardBody(state.createRewardAmount)),
				_Utils_Tuple2(
				'participation',
				$author$project$Main$createParticipationBody(state)),
				_Utils_Tuple2(
				'visibility',
				$author$project$Main$createVisibilityBody(state)),
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
				$elm$json$Json$Encode$string('{\"kind\":\"freeform\"}')),
				_Utils_Tuple2(
				'payload',
				$elm$json$Json$Encode$object(
					_List_fromArray(
						[
							_Utils_Tuple2(
							'kind',
							$elm$json$Json$Encode$string('none')),
							_Utils_Tuple2(
							'json',
							$elm$json$Json$Encode$string(''))
						])))
			]));
};
var $author$project$Main$taskDetailFromResponse = function (response) {
	return {assigneeScope: response.assigneeScope, availabilityKind: response.availabilityKind, createdBy: response.createdBy, description: response.description, id: response.id, participationPolicy: response.participationPolicy, reservationExpiryHours: response.reservationExpiryHours, responseSchemaJson: response.responseSchemaJSON, rewardCollectibleCount: response.rewardCollectibleCount, rewardCreditAmount: response.rewardCreditAmount, rewardKind: response.rewardKind, state: response.state, title: response.title, viewerAction: response.viewerAction};
};
var $author$project$Sharecrop$Generated$Task$TaskResponse = function (id) {
	return function (ownerKind) {
		return function (ownerID) {
			return function (title) {
				return function (description) {
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
																return function (seriesKind) {
																	return function (seriesID) {
																		return function (seriesPosition) {
																			return function (responseSchemaJSON) {
																				return function (payloadKind) {
																					return function (payloadJSON) {
																						return function (createdBy) {
																							return {assigneeScope: assigneeScope, availabilityKind: availabilityKind, createdBy: createdBy, description: description, id: id, ownerID: ownerID, ownerKind: ownerKind, participationPolicy: participationPolicy, payloadJSON: payloadJSON, payloadKind: payloadKind, reservationExpiryHours: reservationExpiryHours, responseSchemaJSON: responseSchemaJSON, rewardCollectibleCount: rewardCollectibleCount, rewardCreditAmount: rewardCreditAmount, rewardKind: rewardKind, seriesID: seriesID, seriesKind: seriesKind, seriesPosition: seriesPosition, state: state, title: title, viewerAction: viewerAction, visibilityID: visibilityID, visibilityKind: visibilityKind};
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
var $elm$json$Json$Decode$map7 = _Json_map7;
var $elm$json$Json$Decode$map8 = _Json_map8;
var $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeOrganizationTeam = {$: 'TaskAssigneeScopeOrganizationTeam'};
var $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser = {$: 'TaskAssigneeScopeUser'};
var $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder = A2(
	$elm$json$Json$Decode$andThen,
	function (value) {
		switch (value) {
			case 'user':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser);
			case 'organization_team':
				return $elm$json$Json$Decode$succeed($author$project$Sharecrop$Generated$Task$TaskAssigneeScopeOrganizationTeam);
			default:
				return $elm$json$Json$Decode$fail('invalid TaskAssigneeScope');
		}
	},
	$elm$json$Json$Decode$string);
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
		return A8(
			$elm$json$Json$Decode$map7,
			finish,
			A2($elm$json$Json$Decode$field, 'series_kind', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'series_id', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'series_position', $elm$json$Json$Decode$int),
			A2($elm$json$Json$Decode$field, 'response_schema_json', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'payload_kind', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'payload_json', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string));
	},
	A2(
		$elm$json$Json$Decode$andThen,
		function (finish) {
			return A9(
				$elm$json$Json$Decode$map8,
				finish,
				A2($elm$json$Json$Decode$field, 'participation_policy', $author$project$Sharecrop$Generated$Task$taskParticipationPolicyDecoder),
				A2($elm$json$Json$Decode$field, 'assignee_scope', $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder),
				A2($elm$json$Json$Decode$field, 'reservation_expiry_hours', $elm$json$Json$Decode$int),
				A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Task$taskStateDecoder),
				A2($elm$json$Json$Decode$field, 'visibility_kind', $author$project$Sharecrop$Generated$Task$taskVisibilityKindDecoder),
				A2($elm$json$Json$Decode$field, 'visibility_id', $elm$json$Json$Decode$string),
				A2($elm$json$Json$Decode$field, 'availability_kind', $author$project$Sharecrop$Generated$Task$taskAvailabilityKindDecoder),
				A2($elm$json$Json$Decode$field, 'viewer_action', $author$project$Sharecrop$Generated$Task$taskViewerActionDecoder));
		},
		A9(
			$elm$json$Json$Decode$map8,
			$author$project$Sharecrop$Generated$Task$TaskResponse,
			A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'owner_kind', $author$project$Sharecrop$Generated$Task$taskOwnerKindDecoder),
			A2($elm$json$Json$Decode$field, 'owner_id', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'title', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'description', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'reward_kind', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'reward_credit_amount', $elm$json$Json$Decode$int),
			A2($elm$json$Json$Decode$field, 'reward_collectible_count', $elm$json$Json$Decode$int))));
var $author$project$Main$taskDetailDecoder = A2($elm$json$Json$Decode$map, $author$project$Main$taskDetailFromResponse, $author$project$Sharecrop$Generated$Task$taskResponseDecoder);
var $author$project$Main$postCreateTask = function (state) {
	return A5(
		$author$project$Main$authorizedRequest,
		'POST',
		state.accessToken,
		'/api/tasks',
		$elm$http$Http$jsonBody(
			$author$project$Main$createTaskRequestBody(state)),
		A2($elm$http$Http$expectJson, $author$project$Main$CreateTaskReceived, $author$project$Main$taskDetailDecoder));
};
var $author$project$Main$createTaskCommand = F2(
	function (model, state) {
		return ($elm$core$String$isEmpty(
			$elm$core$String$trim(state.createTitle)) || $elm$core$String$isEmpty(
			$elm$core$String$trim(state.createDescription))) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							createMessage: $elm$core$Maybe$Just('Title and description are required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : ((($author$project$Main$reservationHoursValue(state.createReservationHours) < 1) || ($author$project$Main$reservationHoursValue(state.createReservationHours) > 720)) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
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
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{createMessage: $elm$core$Maybe$Nothing});
				}),
			$author$project$Main$postCreateTask(state)));
	});
var $author$project$Main$credentialsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.credentials;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$enterPage = F2(
	function (page, state) {
		switch (page.$) {
			case 'OrganizationDetailPage':
				var organizationId = page.a;
				return _Utils_update(
					state,
					{activeOrgId: organizationId, orgBalance: $elm$core$Maybe$Nothing, orgMembers: _List_Nil, orgTasks: _List_Nil, orgTeamMessage: $elm$core$Maybe$Nothing, orgTeams: _List_Nil, page: page, provisionMemberMessage: $elm$core$Maybe$Nothing});
			case 'UserDetailPage':
				return _Utils_update(
					state,
					{page: page, userProfile: $elm$core$Maybe$Nothing});
			case 'UserWorkPage':
				return _Utils_update(
					state,
					{page: page, userWork: _List_Nil});
			case 'UserSubmissionsPage':
				return _Utils_update(
					state,
					{page: page, userSubmissions: _List_Nil});
			case 'SeriesDetailPage':
				return _Utils_update(
					state,
					{page: page, seriesDetail: $elm$core$Maybe$Nothing});
			case 'TeamDetailPage':
				return _Utils_update(
					state,
					{page: page, teamDetail: $elm$core$Maybe$Nothing});
			default:
				return _Utils_update(
					state,
					{page: page});
		}
	});
var $author$project$Main$entriesFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.entries;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$CollectiblesReceived = function (a) {
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
var $author$project$Main$fetchCollectibles = function (token) {
	return A5(
		$author$project$Main$authorizedRequest,
		'GET',
		token,
		'/api/collectibles',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Main$CollectiblesReceived, $author$project$Sharecrop$Generated$Collectible$collectiblesResponseDecoder));
};
var $author$project$Main$DiscoveryReceived = function (a) {
	return {$: 'DiscoveryReceived', a: a};
};
var $author$project$Main$boolQuery = function (value) {
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
													return function (createdBy) {
														return function (activeAssigneeKind) {
															return function (activeAssigneeID) {
																return {activeAssigneeID: activeAssigneeID, activeAssigneeKind: activeAssigneeKind, assigneeScope: assigneeScope, availabilityKind: availabilityKind, createdBy: createdBy, id: id, ownerKind: ownerKind, participationPolicy: participationPolicy, reservationExpiryHours: reservationExpiryHours, rewardCollectibleCount: rewardCollectibleCount, rewardCreditAmount: rewardCreditAmount, rewardKind: rewardKind, state: state, title: title, viewerAction: viewerAction, visibilityKind: visibilityKind};
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
		return A9(
			$elm$json$Json$Decode$map8,
			finish,
			A2($elm$json$Json$Decode$field, 'reservation_expiry_hours', $elm$json$Json$Decode$int),
			A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Task$taskStateDecoder),
			A2($elm$json$Json$Decode$field, 'visibility_kind', $author$project$Sharecrop$Generated$Task$taskVisibilityKindDecoder),
			A2($elm$json$Json$Decode$field, 'availability_kind', $author$project$Sharecrop$Generated$Task$taskAvailabilityKindDecoder),
			A2($elm$json$Json$Decode$field, 'viewer_action', $author$project$Sharecrop$Generated$Task$taskViewerActionDecoder),
			A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'active_assignee_kind', $elm$json$Json$Decode$string),
			A2($elm$json$Json$Decode$field, 'active_assignee_id', $elm$json$Json$Decode$string));
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
		A2($elm$json$Json$Decode$field, 'assignee_scope', $author$project$Sharecrop$Generated$Task$taskAssigneeScopeDecoder)));
var $author$project$Sharecrop$Generated$Task$tasksResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Task$TasksResponse,
	A2(
		$elm$json$Json$Decode$field,
		'tasks',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Task$taskListItemResponseDecoder)));
var $author$project$Main$fetchDiscovery = F2(
	function (token, includeReserved) {
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			'/api/tasks?scope=public&include_reserved=' + $author$project$Main$boolQuery(includeReserved),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$DiscoveryReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
	});
var $author$project$Main$OrgTeamsReceived = function (a) {
	return {$: 'OrgTeamsReceived', a: a};
};
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
var $author$project$Main$fetchOrgTeams = F2(
	function (token, organizationId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			'/api/organizations/' + (organizationId + '/teams'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$OrgTeamsReceived, $author$project$Sharecrop$Generated$Team$teamsResponseDecoder));
	});
var $author$project$Main$TasksReceived = function (a) {
	return {$: 'TasksReceived', a: a};
};
var $author$project$Main$fetchTasks = F2(
	function (token, stateFilter) {
		var query = (stateFilter === '') ? '/api/tasks?scope=user' : ('/api/tasks?scope=user&state=' + stateFilter);
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			query,
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$TasksReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
	});
var $author$project$Main$escrowStateLabel = function (state) {
	switch (state.$) {
		case 'EscrowStateHeld':
			return 'held';
		case 'EscrowStateReleased':
			return 'released';
		default:
			return 'refunded';
	}
};
var $author$project$Main$fundSuccessLabel = function (escrow) {
	return 'Escrowed ' + ($elm$core$String$fromInt(escrow.amount) + (' credits (' + ($author$project$Main$escrowStateLabel(escrow.state) + ').')));
};
var $author$project$Main$FundReceived = function (a) {
	return {$: 'FundReceived', a: a};
};
var $author$project$Main$fundingRequestBody = F3(
	function (taskId, amount, organizationId) {
		return $elm$json$Json$Encode$object(
			_List_fromArray(
				[
					_Utils_Tuple2(
					'amount',
					$elm$json$Json$Encode$int(amount)),
					_Utils_Tuple2(
					'idempotency_key',
					$elm$json$Json$Encode$string('fund:' + taskId)),
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
var $author$project$Main$postFunding = F4(
	function (token, taskId, amount, organizationId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/funding'),
			$elm$http$Http$jsonBody(
				A3($author$project$Main$fundingRequestBody, taskId, amount, organizationId)),
			A2($elm$http$Http$expectJson, $author$project$Main$FundReceived, $author$project$Sharecrop$Generated$Ledger$taskEscrowResponseDecoder));
	});
var $author$project$Main$fundTaskCommand = F2(
	function (model, state) {
		var _v0 = $elm$core$String$toInt(state.fundAmount);
		if (_v0.$ === 'Just') {
			var amount = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{fundMessage: $elm$core$Maybe$Nothing});
					}),
				A4($author$project$Main$postFunding, state.accessToken, state.fundTaskId, amount, state.fundOrganizationId));
		} else {
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
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
var $author$project$Main$httpErrorLabel = function (error) {
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
var $author$project$Main$BalanceReceived = function (a) {
	return {$: 'BalanceReceived', a: a};
};
var $author$project$Sharecrop$Generated$Ledger$BalanceResponse = function (amount) {
	return {amount: amount};
};
var $author$project$Sharecrop$Generated$Ledger$balanceResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Ledger$BalanceResponse,
	A2($elm$json$Json$Decode$field, 'amount', $elm$json$Json$Decode$int));
var $author$project$Main$fetchBalance = function (token) {
	return A5(
		$author$project$Main$authorizedRequest,
		'GET',
		token,
		'/api/credits/balance',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Main$BalanceReceived, $author$project$Sharecrop$Generated$Ledger$balanceResponseDecoder));
};
var $author$project$Main$CredentialsReceived = function (a) {
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
var $author$project$Main$fetchCredentials = function (token) {
	return A5(
		$author$project$Main$authorizedRequest,
		'GET',
		token,
		'/api/agent-credentials',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Main$CredentialsReceived, $author$project$Sharecrop$Generated$Agent$agentCredentialsResponseDecoder));
};
var $author$project$Main$LedgerReceived = function (a) {
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
var $author$project$Main$fetchLedger = function (token) {
	return A5(
		$author$project$Main$authorizedRequest,
		'GET',
		token,
		'/api/credits/ledger',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Main$LedgerReceived, $author$project$Sharecrop$Generated$Ledger$ledgerResponseDecoder));
};
var $author$project$Main$OrganizationsReceived = function (a) {
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
var $author$project$Main$fetchOrganizations = function (token) {
	return A5(
		$author$project$Main$authorizedRequest,
		'GET',
		token,
		'/api/organizations',
		$elm$http$Http$emptyBody,
		A2($elm$http$Http$expectJson, $author$project$Main$OrganizationsReceived, $author$project$Sharecrop$Generated$Organization$organizationsResponseDecoder));
};
var $author$project$Main$loadAfterAuth = function (token) {
	return $elm$core$Platform$Cmd$batch(
		_List_fromArray(
			[
				$author$project$Main$fetchBalance(token),
				$author$project$Main$fetchLedger(token),
				A2($author$project$Main$fetchTasks, token, ''),
				$author$project$Main$fetchCredentials(token),
				$author$project$Main$fetchCollectibles(token),
				$author$project$Main$fetchOrganizations(token)
			]));
};
var $author$project$Main$participationPolicyTag = function (policy) {
	switch (policy.$) {
		case 'TaskParticipationPolicyOpen':
			return 'open';
		case 'TaskParticipationPolicyReservationRequired':
			return 'reservation_required';
		default:
			return 'approval_required';
	}
};
var $author$project$Main$visibilityDefaultTag = 'default';
var $author$project$Main$emptyLoggedIn = function (response) {
	return {
		accessToken: response.accessToken,
		activeOrgId: '',
		agentLabel: '',
		agentMessage: $elm$core$Maybe$Nothing,
		agentScopes: _List_fromArray(
			[$author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite]),
		awardMessage: $elm$core$Maybe$Nothing,
		awardTaskId: '',
		balance: $elm$core$Maybe$Nothing,
		collectibleKind: $author$project$Sharecrop$Generated$Collectible$CollectibleKindBadge,
		collectibleMessage: $elm$core$Maybe$Nothing,
		collectibleName: '',
		collectiblePolicy: $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyNonTransferableExceptPayout,
		collectibles: _List_Nil,
		createAssigneeScope: $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser,
		createDescription: '',
		createMessage: $elm$core$Maybe$Nothing,
		createOrgName: '',
		createOrgTeamName: '',
		createParticipationPolicy: $author$project$Main$participationPolicyTag($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen),
		createReservationHours: '48',
		createRewardAmount: '',
		createScopeOrganizationId: '',
		createScopeTeamId: '',
		createScopeUserId: '',
		createTaskOwner: '',
		createTitle: '',
		createVisibility: $author$project$Main$visibilityDefaultTag,
		credentials: _List_Nil,
		detail: $elm$core$Maybe$Nothing,
		discoveryIncludeReserved: false,
		discoveryTasks: _List_Nil,
		entries: _List_Nil,
		fundAmount: '',
		fundMessage: $elm$core$Maybe$Nothing,
		fundOrganizationId: '',
		fundTaskId: '',
		newCredential: $elm$core$Maybe$Nothing,
		orgBalance: $elm$core$Maybe$Nothing,
		orgMembers: _List_Nil,
		orgMessage: $elm$core$Maybe$Nothing,
		orgTasks: _List_Nil,
		orgTeamMessage: $elm$core$Maybe$Nothing,
		orgTeams: _List_Nil,
		organizations: _List_Nil,
		page: $author$project$Main$OverviewPage,
		provisionMemberEmail: '',
		provisionMemberMessage: $elm$core$Maybe$Nothing,
		reservationMessage: $elm$core$Maybe$Nothing,
		reservations: _List_Nil,
		reviewBan: false,
		reviewMessage: $elm$core$Maybe$Nothing,
		reviewNote: '',
		reviewPartialCredit: '',
		reviewTip: '',
		seriesDetail: $elm$core$Maybe$Nothing,
		subjectId: response.subjectID,
		submissions: _List_Nil,
		submitInput: '',
		submitMessage: $elm$core$Maybe$Nothing,
		taskStateFilter: '',
		tasks: _List_Nil,
		teamDetail: $elm$core$Maybe$Nothing,
		userProfile: $elm$core$Maybe$Nothing,
		userSubmissions: _List_Nil,
		userWork: _List_Nil
	};
};
var $author$project$Main$loggedInForPage = F2(
	function (response, page) {
		var state = $author$project$Main$emptyLoggedIn(response);
		return _Utils_update(
			state,
			{page: page});
	});
var $author$project$Main$membersFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.members;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$MintReceived = function (a) {
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
var $author$project$Main$collectibleRequestBody = F3(
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
var $author$project$Main$postCollectible = F4(
	function (token, name, kind, policy) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/collectibles',
			$elm$http$Http$jsonBody(
				A3($author$project$Main$collectibleRequestBody, name, kind, policy)),
			A2($elm$http$Http$expectJson, $author$project$Main$MintReceived, $author$project$Sharecrop$Generated$Collectible$collectibleResponseDecoder));
	});
var $author$project$Main$mintCommand = F2(
	function (model, state) {
		return $elm$core$String$isEmpty(
			$elm$core$String$trim(state.collectibleName)) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
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
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{collectibleMessage: $elm$core$Maybe$Nothing});
				}),
			A4($author$project$Main$postCollectible, state.accessToken, state.collectibleName, state.collectibleKind, state.collectiblePolicy));
	});
var $author$project$Main$mintSuccessLabel = function (collectible) {
	return 'Minted ' + (collectible.name + (' (' + ($author$project$Main$collectibleStateLabel(collectible.state) + ').')));
};
var $author$project$Sharecrop$Generated$Organization$OrganizationMembersResponse = function (members) {
	return {members: members};
};
var $author$project$Sharecrop$Generated$Organization$OrganizationMemberResponse = F5(
	function (id, organizationID, userID, status, roles) {
		return {id: id, organizationID: organizationID, roles: roles, status: status, userID: userID};
	});
var $elm$json$Json$Decode$map5 = _Json_map5;
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
var $author$project$Main$organizationsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.organizations;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$AuthReceived = function (a) {
	return {$: 'AuthReceived', a: a};
};
var $author$project$Main$authRequestBody = function (model) {
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
var $author$project$Main$postAuth = F2(
	function (url, model) {
		return $elm$http$Http$post(
			{
				body: $elm$http$Http$jsonBody(
					$author$project$Main$authRequestBody(model)),
				expect: A2($elm$http$Http$expectJson, $author$project$Main$AuthReceived, $author$project$Sharecrop$Generated$Auth$authResponseDecoder),
				url: url
			});
	});
var $author$project$Main$LogoutReceived = function (a) {
	return {$: 'LogoutReceived', a: a};
};
var $author$project$Main$postLogout = $elm$http$Http$post(
	{
		body: $elm$http$Http$emptyBody,
		expect: $elm$http$Http$expectWhatever($author$project$Main$LogoutReceived),
		url: '/api/auth/logout'
	});
var $author$project$Main$OpenTaskReceived = function (a) {
	return {$: 'OpenTaskReceived', a: a};
};
var $author$project$Main$postOpenTask = F2(
	function (token, taskId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/open'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Main$OpenTaskReceived, $author$project$Main$taskDetailDecoder));
	});
var $author$project$Main$RefundTaskReceived = function (a) {
	return {$: 'RefundTaskReceived', a: a};
};
var $author$project$Main$postRefundTask = F2(
	function (token, taskId) {
		return A5(
			$author$project$Main$authorizedRequest,
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
			A2($elm$http$Http$expectJson, $author$project$Main$RefundTaskReceived, $author$project$Sharecrop$Generated$Ledger$taskEscrowResponseDecoder));
	});
var $author$project$Main$ReservationReceived = function (a) {
	return {$: 'ReservationReceived', a: a};
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
var $author$project$Main$postReservation = F2(
	function (token, taskId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/reservations'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Main$ReservationReceived, $author$project$Sharecrop$Generated$Task$taskReservationResponseDecoder));
	});
var $author$project$Main$ProvisionMemberReceived = function (a) {
	return {$: 'ProvisionMemberReceived', a: a};
};
var $author$project$Main$provisionMemberCommand = F2(
	function (model, state) {
		return ($elm$core$String$isEmpty(
			$elm$core$String$trim(state.provisionMemberEmail)) || (state.activeOrgId === '')) ? _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{
							provisionMemberMessage: $elm$core$Maybe$Just('A member email is required.')
						});
				}),
			$elm$core$Platform$Cmd$none) : _Utils_Tuple2(
			A2(
				$author$project$Main$updateLoggedIn,
				model,
				function (current) {
					return _Utils_update(
						current,
						{provisionMemberMessage: $elm$core$Maybe$Nothing});
				}),
			A5(
				$author$project$Main$authorizedRequest,
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
								A2(
									$elm$json$Json$Encode$list,
									$elm$json$Json$Encode$string,
									_List_fromArray(
										['member'])))
							]))),
				$elm$http$Http$expectWhatever($author$project$Main$ProvisionMemberReceived)));
	});
var $elm$browser$Browser$Navigation$pushUrl = _Browser_pushUrl;
var $author$project$Main$SubmissionsReceived = function (a) {
	return {$: 'SubmissionsReceived', a: a};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionsResponse = function (submissions) {
	return {submissions: submissions};
};
var $author$project$Sharecrop$Generated$Submission$SubmissionResponse = F7(
	function (id, taskID, submitterID, state, responseJSON, reviewNote, validationErrors) {
		return {id: id, responseJSON: responseJSON, reviewNote: reviewNote, state: state, submitterID: submitterID, taskID: taskID, validationErrors: validationErrors};
	});
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
var $author$project$Sharecrop$Generated$Submission$submissionResponseDecoder = A8(
	$elm$json$Json$Decode$map7,
	$author$project$Sharecrop$Generated$Submission$SubmissionResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'task_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'submitter_id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'state', $author$project$Sharecrop$Generated$Submission$submissionStateDecoder),
	A2($elm$json$Json$Decode$field, 'response_json', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'review_note', $elm$json$Json$Decode$string),
	A2(
		$elm$json$Json$Decode$field,
		'validation_errors',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionValidationErrorResponseDecoder)));
var $author$project$Sharecrop$Generated$Submission$submissionsResponseDecoder = A2(
	$elm$json$Json$Decode$map,
	$author$project$Sharecrop$Generated$Submission$SubmissionsResponse,
	A2(
		$elm$json$Json$Decode$field,
		'submissions',
		$elm$json$Json$Decode$list($author$project$Sharecrop$Generated$Submission$submissionResponseDecoder)));
var $author$project$Main$fetchSubmissions = F2(
	function (token, taskId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + (taskId + '/submissions'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$SubmissionsReceived, $author$project$Sharecrop$Generated$Submission$submissionsResponseDecoder));
	});
var $author$project$Main$refreshAfterAccept = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		var _v1 = state.page;
		if (_v1.$ === 'TaskDetailPage') {
			var taskId = _v1.a;
			return $elm$core$Platform$Cmd$batch(
				_List_fromArray(
					[
						A2($author$project$Main$fetchSubmissions, state.accessToken, taskId),
						$author$project$Main$fetchBalance(state.accessToken)
					]));
		} else {
			return $elm$core$Platform$Cmd$none;
		}
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshCollectibles = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $author$project$Main$fetchCollectibles(state.accessToken);
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshCredentials = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $author$project$Main$fetchCredentials(state.accessToken);
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$DetailReceived = function (a) {
	return {$: 'DetailReceived', a: a};
};
var $author$project$Main$publicTaskDetailFromResponse = function (response) {
	return $author$project$Main$taskDetailFromResponse(response);
};
var $author$project$Main$publicTaskDetailDecoder = A2($elm$json$Json$Decode$map, $author$project$Main$publicTaskDetailFromResponse, $author$project$Sharecrop$Generated$Task$taskResponseDecoder);
var $author$project$Main$fetchPublicTaskDetail = F2(
	function (token, taskId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + taskId,
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$DetailReceived, $author$project$Main$publicTaskDetailDecoder));
	});
var $author$project$Main$ReservationsReceived = function (a) {
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
var $author$project$Main$fetchReservations = F2(
	function (token, taskId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			'/api/tasks/' + (taskId + '/reservations'),
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$ReservationsReceived, $author$project$Sharecrop$Generated$Task$taskReservationsResponseDecoder));
	});
var $author$project$Main$refreshDetailReservations = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		var _v1 = state.page;
		if (_v1.$ === 'TaskDetailPage') {
			var taskId = _v1.a;
			return $elm$core$Platform$Cmd$batch(
				_List_fromArray(
					[
						A2($author$project$Main$fetchPublicTaskDetail, state.accessToken, taskId),
						A2($author$project$Main$fetchReservations, state.accessToken, taskId)
					]));
		} else {
			return $elm$core$Platform$Cmd$none;
		}
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshDetailSubmissions = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		var _v1 = state.page;
		if (_v1.$ === 'TaskDetailPage') {
			var taskId = _v1.a;
			return A2($author$project$Main$fetchSubmissions, state.accessToken, taskId);
		} else {
			return $elm$core$Platform$Cmd$none;
		}
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshLedger = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					$author$project$Main$fetchBalance(state.accessToken),
					$author$project$Main$fetchLedger(state.accessToken)
				]));
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshOrganizations = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $author$project$Main$fetchOrganizations(state.accessToken);
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshTasksAndDiscovery = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A2($author$project$Main$fetchTasks, state.accessToken, state.taskStateFilter),
					A2($author$project$Main$fetchDiscovery, state.accessToken, state.discoveryIncludeReserved)
				]));
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $author$project$Main$refreshTasksAndLedger = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedIn') {
		var state = _v0.a;
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A2($author$project$Main$fetchTasks, state.accessToken, state.taskStateFilter),
					$author$project$Main$fetchBalance(state.accessToken),
					$author$project$Main$fetchLedger(state.accessToken)
				]));
	} else {
		return $elm$core$Platform$Cmd$none;
	}
};
var $elm$json$Json$Encode$bool = _Json_wrap;
var $author$project$Main$rejectRequestBody = F5(
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
						$author$project$Main$intInputOrZero(partialCredit))),
					_Utils_Tuple2(
					'tip_amount',
					$elm$json$Json$Encode$int(
						$author$project$Main$intInputOrZero(tipAmount))),
					_Utils_Tuple2(
					'ban_implementor',
					$elm$json$Json$Encode$bool(banImplementor))
				]));
	});
var $author$project$Main$postReject = F7(
	function (token, taskId, submissionId, reviewNote, partialCredit, tipAmount, banImplementor) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/submissions/' + (submissionId + '/reject'))),
			$elm$http$Http$jsonBody(
				A5($author$project$Main$rejectRequestBody, submissionId, reviewNote, partialCredit, tipAmount, banImplementor)),
			$elm$http$Http$expectWhatever($author$project$Main$ReviewActionReceived));
	});
var $author$project$Main$rejectCommand = F3(
	function (model, state, submissionId) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reviewMessage: $elm$core$Maybe$Nothing});
					}),
				A7($author$project$Main$postReject, state.accessToken, taskId, submissionId, state.reviewNote, state.reviewPartialCredit, state.reviewTip, state.reviewBan));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Main$requestChangesBody = function (reviewNote) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'review_note',
				$elm$json$Json$Encode$string(reviewNote))
			]));
};
var $author$project$Main$postRequestChanges = F4(
	function (token, taskId, submissionId, reviewNote) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/submissions/' + (submissionId + '/request-changes'))),
			$elm$http$Http$jsonBody(
				$author$project$Main$requestChangesBody(reviewNote)),
			$elm$http$Http$expectWhatever($author$project$Main$ReviewActionReceived));
	});
var $author$project$Main$requestChangesCommand = F3(
	function (model, state, submissionId) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reviewMessage: $elm$core$Maybe$Nothing});
					}),
				A4($author$project$Main$postRequestChanges, state.accessToken, taskId, submissionId, state.reviewNote));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Main$ReservationChangeReceived = function (a) {
	return {$: 'ReservationChangeReceived', a: a};
};
var $author$project$Main$postReservationChange = F4(
	function (token, taskId, reservationId, action) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + ('/reservations/' + (reservationId + ('/' + action)))),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Main$ReservationChangeReceived, $author$project$Sharecrop$Generated$Task$taskReservationResponseDecoder));
	});
var $author$project$Main$reservationChangeCommand = F4(
	function (model, state, reservationId, action) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{reservationMessage: $elm$core$Maybe$Nothing});
					}),
				A4($author$project$Main$postReservationChange, state.accessToken, taskId, reservationId, action));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Main$reservationStateLabel = function (state) {
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
var $author$project$Main$reservationSuccessLabel = function (reservation) {
	return 'Reservation ' + ($author$project$Main$reservationStateLabel(reservation.state) + '.');
};
var $author$project$Main$AgentRevoked = function (a) {
	return {$: 'AgentRevoked', a: a};
};
var $author$project$Main$revokeAgent = F2(
	function (token, credentialId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/agent-credentials/' + (credentialId + '/revoke'),
			$elm$http$Http$jsonBody(
				$elm$json$Json$Encode$object(_List_Nil)),
			A2($elm$http$Http$expectJson, $author$project$Main$AgentRevoked, $author$project$Sharecrop$Generated$Agent$agentCredentialResponseDecoder));
	});
var $author$project$Main$SeriesDetailReceived = function (a) {
	return {$: 'SeriesDetailReceived', a: a};
};
var $author$project$Main$TeamDetailReceived = function (a) {
	return {$: 'TeamDetailReceived', a: a};
};
var $author$project$Main$UserSubmissionsReceived = function (a) {
	return {$: 'UserSubmissionsReceived', a: a};
};
var $author$project$Main$UserWorkReceived = function (a) {
	return {$: 'UserWorkReceived', a: a};
};
var $author$project$Main$fetchDetailCommands = F2(
	function (token, taskId) {
		return $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A2($author$project$Main$fetchPublicTaskDetail, token, taskId),
					A2($author$project$Main$fetchSubmissions, token, taskId),
					A2($author$project$Main$fetchReservations, token, taskId)
				]));
	});
var $author$project$Main$UserProfileReceived = function (a) {
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
var $author$project$Main$fetchUserProfile = F2(
	function (token, userId) {
		return A5(
			$author$project$Main$authorizedRequest,
			'GET',
			token,
			'/api/users/' + userId,
			$elm$http$Http$emptyBody,
			A2($elm$http$Http$expectJson, $author$project$Main$UserProfileReceived, $author$project$Sharecrop$Generated$Task$userProfileResponseDecoder));
	});
var $author$project$Main$OrgBalanceReceived = function (a) {
	return {$: 'OrgBalanceReceived', a: a};
};
var $author$project$Main$OrgTasksReceived = function (a) {
	return {$: 'OrgTasksReceived', a: a};
};
var $author$project$Main$loadOrganization = F2(
	function (token, organizationId) {
		return (organizationId === '') ? $elm$core$Platform$Cmd$none : $elm$core$Platform$Cmd$batch(
			_List_fromArray(
				[
					A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/organizations/' + (organizationId + '/credits/balance'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$OrgBalanceReceived, $author$project$Sharecrop$Generated$Ledger$balanceResponseDecoder)),
					A2($author$project$Main$fetchOrgTeams, token, organizationId),
					A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/organizations/' + (organizationId + '/members'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$OrgMembersReceived, $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder)),
					A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/tasks?scope=organization&organization_id=' + organizationId,
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$OrgTasksReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder))
				]));
	});
var $author$project$Sharecrop$Generated$TaskSeries$TaskSeriesResponse = F4(
	function (id, ownerKind, title, createdBy) {
		return {createdBy: createdBy, id: id, ownerKind: ownerKind, title: title};
	});
var $author$project$Sharecrop$Generated$TaskSeries$taskSeriesResponseDecoder = A5(
	$elm$json$Json$Decode$map4,
	$author$project$Sharecrop$Generated$TaskSeries$TaskSeriesResponse,
	A2($elm$json$Json$Decode$field, 'id', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'owner_kind', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'title', $elm$json$Json$Decode$string),
	A2($elm$json$Json$Decode$field, 'created_by', $elm$json$Json$Decode$string));
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
var $author$project$Main$routeLoadCmd = F2(
	function (token, page) {
		switch (page.$) {
			case 'OverviewPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Main$fetchBalance(token),
							$author$project$Main$fetchLedger(token)
						]));
			case 'TasksPage':
				return A2($author$project$Main$fetchTasks, token, '');
			case 'CreateTaskPage':
				return $author$project$Main$fetchOrganizations(token);
			case 'TaskDetailPage':
				var taskId = page.a;
				return A2($author$project$Main$fetchDetailCommands, token, taskId);
			case 'DiscoveryPage':
				return A2($author$project$Main$fetchDiscovery, token, false);
			case 'FundingPage':
				return A2($author$project$Main$fetchTasks, token, '');
			case 'AgentsPage':
				return $author$project$Main$fetchCredentials(token);
			case 'CollectiblesPage':
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Main$fetchCollectibles(token),
							A2($author$project$Main$fetchTasks, token, '')
						]));
			case 'OrganizationsPage':
				return $author$project$Main$fetchOrganizations(token);
			case 'OrganizationDetailPage':
				var organizationId = page.a;
				return $elm$core$Platform$Cmd$batch(
					_List_fromArray(
						[
							$author$project$Main$fetchOrganizations(token),
							A2($author$project$Main$loadOrganization, token, organizationId)
						]));
			case 'UserDetailPage':
				var userId = page.a;
				return A2($author$project$Main$fetchUserProfile, token, userId);
			case 'UserWorkPage':
				var userId = page.a;
				return A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/users/' + (userId + '/work'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$UserWorkReceived, $author$project$Sharecrop$Generated$Task$tasksResponseDecoder));
			case 'UserSubmissionsPage':
				var userId = page.a;
				return A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/users/' + (userId + '/submissions'),
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$UserSubmissionsReceived, $author$project$Sharecrop$Generated$Submission$submissionsResponseDecoder));
			case 'CollectibleDetailPage':
				return $author$project$Main$fetchCollectibles(token);
			case 'SeriesDetailPage':
				var seriesId = page.a;
				return A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/task-series/' + seriesId,
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$SeriesDetailReceived, $author$project$Sharecrop$Generated$TaskSeries$taskSeriesResponseDecoder));
			default:
				var teamId = page.a;
				return A5(
					$author$project$Main$authorizedRequest,
					'GET',
					token,
					'/api/teams/' + teamId,
					$elm$http$Http$emptyBody,
					A2($elm$http$Http$expectJson, $author$project$Main$TeamDetailReceived, $author$project$Sharecrop$Generated$Team$teamDetailResponseDecoder));
		}
	});
var $author$project$Main$submissionsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.submissions;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$SubmitReceived = function (a) {
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
var $author$project$Main$submissionRequestBody = function (responseJson) {
	return $elm$json$Json$Encode$object(
		_List_fromArray(
			[
				_Utils_Tuple2(
				'response_json',
				$elm$json$Json$Encode$string(responseJson))
			]));
};
var $author$project$Main$postSubmission = F3(
	function (token, taskId, responseJson) {
		return A5(
			$author$project$Main$authorizedRequest,
			'POST',
			token,
			'/api/tasks/' + (taskId + '/submissions'),
			$elm$http$Http$jsonBody(
				$author$project$Main$submissionRequestBody(responseJson)),
			A2($elm$http$Http$expectJson, $author$project$Main$SubmitReceived, $author$project$Sharecrop$Generated$Submission$submissionCreatedResponseDecoder));
	});
var $author$project$Main$submitCommand = F2(
	function (model, state) {
		var _v0 = state.page;
		if (_v0.$ === 'TaskDetailPage') {
			var taskId = _v0.a;
			return _Utils_Tuple2(
				A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (current) {
						return _Utils_update(
							current,
							{submitMessage: $elm$core$Maybe$Nothing});
					}),
				A3($author$project$Main$postSubmission, state.accessToken, taskId, state.submitInput));
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
	});
var $author$project$Main$submissionStateLabel = function (state) {
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
var $author$project$Main$submitSuccessLabel = function (created) {
	return 'Submission ' + (created.submission.id + (' (' + ($author$project$Main$submissionStateLabel(created.submission.state) + ').')));
};
var $author$project$Main$tasksFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.tasks;
	} else {
		return _List_Nil;
	}
};
var $author$project$Main$teamsFromResult = function (result) {
	if (result.$ === 'Ok') {
		var response = result.a;
		return response.teams;
	} else {
		return _List_Nil;
	}
};
var $elm$core$Result$toMaybe = function (result) {
	if (result.$ === 'Ok') {
		var v = result.a;
		return $elm$core$Maybe$Just(v);
	} else {
		return $elm$core$Maybe$Nothing;
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
var $elm$core$Basics$neq = _Utils_notEqual;
var $author$project$Main$toggleScope = F2(
	function (scope, scopes) {
		return A2($elm$core$List$member, scope, scopes) ? A2(
			$elm$core$List$filter,
			function (existing) {
				return !_Utils_eq(existing, scope);
			},
			scopes) : A2($elm$core$List$cons, scope, scopes);
	});
var $author$project$Main$withSession = F2(
	function (model, run) {
		var _v0 = model.session;
		if (_v0.$ === 'LoggedIn') {
			var state = _v0.a;
			return run(state);
		} else {
			return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
		}
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
					A2($author$project$Main$postAuth, '/api/auth/register', model));
			case 'LoginClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{authError: $elm$core$Maybe$Nothing}),
					A2($author$project$Main$postAuth, '/api/auth/login', model));
			case 'AuthReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Nothing,
								password: '',
								session: $author$project$Main$LoggedIn(
									A2($author$project$Main$loggedInForPage, response, model.route))
							}),
						$author$project$Main$loadAfterAuth(response.accessToken));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								authError: $elm$core$Maybe$Just(
									$author$project$Main$httpErrorLabel(error))
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
								session: $author$project$Main$LoggedIn(
									A2($author$project$Main$loggedInForPage, response, model.route))
							}),
						$elm$core$Platform$Cmd$batch(
							_List_fromArray(
								[
									$author$project$Main$loadAfterAuth(response.accessToken),
									A2($author$project$Main$routeLoadCmd, response.accessToken, model.route)
								])));
				} else {
					return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
				}
			case 'BalanceReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									balance: $author$project$Main$balanceFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'LedgerReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									entries: $author$project$Main$entriesFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TasksReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									tasks: $author$project$Main$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TaskStateFilterChanged':
				var value = msg.a;
				var updated = A2(
					$author$project$Main$updateLoggedIn,
					model,
					function (state) {
						return _Utils_update(
							state,
							{taskStateFilter: value});
					});
				return A2(
					$author$project$Main$withSession,
					updated,
					function (state) {
						return _Utils_Tuple2(
							updated,
							A2($author$project$Main$fetchTasks, state.accessToken, value));
					});
			case 'CreateTitleChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createDescription: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateRewardAmountChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createRewardAmount: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateVisibilityChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createVisibility: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateScopeUserIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createReservationHours: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateTaskClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$createTaskCommand, model, state);
					});
			case 'CreateTaskReceived':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createDescription: '',
										createMessage: $elm$core$Maybe$Just('Created task ' + created.id),
										createParticipationPolicy: $author$project$Main$participationPolicyTag($author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen),
										createReservationHours: '48',
										createTitle: '',
										fundAmount: (created.rewardKind === 'credit') ? $elm$core$String$fromInt(created.rewardCreditAmount) : state.fundAmount,
										fundTaskId: created.id
									});
							}),
						$author$project$Main$refreshTasksAndLedger(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CredentialsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									credentials: $author$project$Main$credentialsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'FundTaskIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{fundOrganizationId: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'FundClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$fundTaskCommand, model, state);
					});
			case 'FundReceived':
				if (msg.a.$ === 'Ok') {
					var escrow = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										fundMessage: $elm$core$Maybe$Just(
											$author$project$Main$fundSuccessLabel(escrow))
									});
							}),
						$author$project$Main$refreshLedger(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										fundMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OpenTaskClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Main$postOpenTask, state.accessToken, taskId));
					});
			case 'OpenTaskReceived':
				if (msg.a.$ === 'Ok') {
					var detail = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createMessage: $elm$core$Maybe$Just('Task opened.'),
										detail: $elm$core$Maybe$Just(detail)
									});
							}),
						$author$project$Main$refreshTasksAndDiscovery(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'RefundTaskClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Main$postRefundTask, state.accessToken, taskId));
					});
			case 'RefundTaskReceived':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createMessage: $elm$core$Maybe$Just('Task refunded and cancelled.')
									});
							}),
						$author$project$Main$refreshTasksAndLedger(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'AgentLabelChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									agentScopes: A2($author$project$Main$toggleScope, scope, state.agentScopes)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateAgentClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$createAgentCommand, model, state);
					});
			case 'AgentCreated':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										agentMessage: $elm$core$Maybe$Nothing,
										newCredential: $elm$core$Maybe$Just(created)
									});
							}),
						$author$project$Main$refreshCredentials(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										agentMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'RevokeClicked':
				var credentialId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							model,
							A2($author$project$Main$revokeAgent, state.accessToken, credentialId));
					});
			case 'AgentRevoked':
				return _Utils_Tuple2(
					model,
					$author$project$Main$refreshCredentials(model));
			case 'LogoutClicked':
				return _Utils_Tuple2(
					_Utils_update(
						model,
						{email: '', password: '', session: $author$project$Main$LoggedOut}),
					$elm$core$Platform$Cmd$batch(
						_List_fromArray(
							[
								$author$project$Main$postLogout,
								A2($elm$browser$Browser$Navigation$pushUrl, model.key, '/')
							])));
			case 'LogoutReceived':
				return _Utils_Tuple2(model, $elm$core$Platform$Cmd$none);
			case 'DiscoveryIncludeReservedChanged':
				var value = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						var nextState = _Utils_update(
							state,
							{discoveryIncludeReserved: value});
						return _Utils_Tuple2(
							A2(
								$author$project$Main$updateLoggedIn,
								model,
								function (_v1) {
									return nextState;
								}),
							A2($author$project$Main$fetchDiscovery, state.accessToken, value));
					});
			case 'DiscoveryReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									discoveryTasks: $author$project$Main$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'DiscoveryViewClicked':
				var taskId = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (s) {
							return _Utils_update(
								s,
								{detail: $elm$core$Maybe$Nothing, reservationMessage: $elm$core$Maybe$Nothing, reservations: _List_Nil, submissions: _List_Nil, submitInput: '', submitMessage: $elm$core$Maybe$Nothing});
						}),
					A2($elm$browser$Browser$Navigation$pushUrl, model.key, '/tasks/' + taskId));
			case 'DetailReceived':
				if (msg.a.$ === 'Ok') {
					var detail = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										detail: $elm$core$Maybe$Just(detail)
									});
							}),
						$elm$core$Platform$Cmd$none);
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submitMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ReserveClicked':
				var taskId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return _Utils_Tuple2(
							A2(
								$author$project$Main$updateLoggedIn,
								model,
								function (current) {
									return _Utils_update(
										current,
										{reservationMessage: $elm$core$Maybe$Nothing});
								}),
							A2($author$project$Main$postReservation, state.accessToken, taskId));
					});
			case 'ReservationReceived':
				if (msg.a.$ === 'Ok') {
					var reservation = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Main$reservationSuccessLabel(reservation))
									});
							}),
						$author$project$Main$refreshDetailReservations(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ReservationsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
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
							$author$project$Main$updateLoggedIn,
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
					$author$project$Main$withSession,
					model,
					function (state) {
						return A4($author$project$Main$reservationChangeCommand, model, state, reservationId, 'approve');
					});
			case 'DeclineReservationClicked':
				var reservationId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A4($author$project$Main$reservationChangeCommand, model, state, reservationId, 'decline');
					});
			case 'CancelReservationClicked':
				var reservationId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A4($author$project$Main$reservationChangeCommand, model, state, reservationId, 'cancel');
					});
			case 'ReservationChangeReceived':
				if (msg.a.$ === 'Ok') {
					var reservation = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Main$reservationSuccessLabel(reservation))
									});
							}),
						$author$project$Main$refreshDetailReservations(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reservationMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SubmissionsReceived':
				if (msg.a.$ === 'Ok') {
					var response = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
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
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submissions: _List_Nil,
										submitMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'SubmitInputChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{submitInput: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SubmitClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$submitCommand, model, state);
					});
			case 'SubmitReceived':
				if (msg.a.$ === 'Ok') {
					var created = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submitMessage: $elm$core$Maybe$Just(
											$author$project$Main$submitSuccessLabel(created))
									});
							}),
						$author$project$Main$refreshDetailSubmissions(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										submitMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ReviewNoteChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{reviewTip: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ReviewBanChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
					$author$project$Main$withSession,
					model,
					function (state) {
						return A3($author$project$Main$acceptCommand, model, state, submissionId);
					});
			case 'RequestChangesClicked':
				var submissionId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A3($author$project$Main$requestChangesCommand, model, state, submissionId);
					});
			case 'RejectClicked':
				var submissionId = msg.a;
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A3($author$project$Main$rejectCommand, model, state, submissionId);
					});
			case 'ReviewActionReceived':
				if (msg.a.$ === 'Ok') {
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reviewMessage: $elm$core$Maybe$Just('Review saved.')
									});
							}),
						$author$project$Main$refreshAfterAccept(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										reviewMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CollectibleNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{collectiblePolicy: policy});
						}),
					$elm$core$Platform$Cmd$none);
			case 'MintClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$mintCommand, model, state);
					});
			case 'MintReceived':
				if (msg.a.$ === 'Ok') {
					var collectible = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										collectibleMessage: $elm$core$Maybe$Just(
											$author$project$Main$mintSuccessLabel(collectible)),
										collectibleName: ''
									});
							}),
						$author$project$Main$refreshCollectibles(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										collectibleMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CollectiblesReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									collectibles: $author$project$Main$collectiblesFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'AwardTaskIdChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
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
					$author$project$Main$withSession,
					model,
					function (state) {
						return A3($author$project$Main$awardCommand, model, state, collectibleId);
					});
			case 'AwardReceived':
				if (msg.a.$ === 'Ok') {
					var collectible = msg.a.a;
					var updated = A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									awardMessage: $elm$core$Maybe$Just(
										$author$project$Main$awardSuccessLabel(collectible))
								});
						});
					return A2(
						$author$project$Main$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								$elm$core$Platform$Cmd$batch(
									_List_fromArray(
										[
											$author$project$Main$fetchCollectibles(state.accessToken),
											A2($author$project$Main$fetchTasks, state.accessToken, state.taskStateFilter)
										])));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										awardMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrganizationsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									organizations: $author$project$Main$organizationsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createOrgName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$createOrgCommand, model, state);
					});
			case 'CreateOrgReceived':
				if (msg.a.$ === 'Ok') {
					var organization = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										createOrgName: '',
										orgMessage: $elm$core$Maybe$Just('Created organization ' + organization.name)
									});
							}),
						$author$project$Main$refreshOrganizations(model));
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'OrgBalanceReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgBalance: $author$project$Main$balanceFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'OrgTeamsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgTeams: $author$project$Main$teamsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'OrgMembersReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgMembers: $author$project$Main$membersFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserProfileReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									userProfile: $elm$core$Result$toMaybe(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserWorkReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									userWork: $author$project$Main$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'UserSubmissionsReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									userSubmissions: $author$project$Main$submissionsFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'SeriesDetailReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									seriesDetail: $elm$core$Result$toMaybe(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'TeamDetailReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									teamDetail: $elm$core$Result$toMaybe(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'OrgTasksReceived':
				var result = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{
									orgTasks: $author$project$Main$tasksFromResult(result)
								});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgTeamNameChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createOrgTeamName: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'CreateOrgTeamClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$createOrgTeamCommand, model, state);
					});
			case 'CreateOrgTeamReceived':
				if (msg.a.$ === 'Ok') {
					var team = msg.a.a;
					var updated = A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								A2($author$project$Main$fetchOrgTeams, state.accessToken, state.activeOrgId));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										orgTeamMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'ProvisionMemberEmailChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{provisionMemberEmail: value});
						}),
					$elm$core$Platform$Cmd$none);
			case 'ProvisionMemberClicked':
				return A2(
					$author$project$Main$withSession,
					model,
					function (state) {
						return A2($author$project$Main$provisionMemberCommand, model, state);
					});
			case 'ProvisionMemberReceived':
				if (msg.a.$ === 'Ok') {
					var updated = A2(
						$author$project$Main$updateLoggedIn,
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
						$author$project$Main$withSession,
						updated,
						function (state) {
							return _Utils_Tuple2(
								updated,
								A5(
									$author$project$Main$authorizedRequest,
									'GET',
									state.accessToken,
									'/api/organizations/' + (state.activeOrgId + '/members'),
									$elm$http$Http$emptyBody,
									A2($elm$http$Http$expectJson, $author$project$Main$OrgMembersReceived, $author$project$Sharecrop$Generated$Organization$organizationMembersResponseDecoder)));
						});
				} else {
					var error = msg.a.a;
					return _Utils_Tuple2(
						A2(
							$author$project$Main$updateLoggedIn,
							model,
							function (state) {
								return _Utils_update(
									state,
									{
										provisionMemberMessage: $elm$core$Maybe$Just(
											$author$project$Main$httpErrorLabel(error))
									});
							}),
						$elm$core$Platform$Cmd$none);
				}
			case 'CreateTaskOwnerChanged':
				var value = msg.a;
				return _Utils_Tuple2(
					A2(
						$author$project$Main$updateLoggedIn,
						model,
						function (state) {
							return _Utils_update(
								state,
								{createTaskOwner: value});
						}),
					$elm$core$Platform$Cmd$none);
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
			default:
				var url = msg.a;
				var page = $author$project$Main$pageFromUrl(url);
				var _v3 = model.session;
				if (_v3.$ === 'LoggedIn') {
					var state = _v3.a;
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{
								route: page,
								session: $author$project$Main$LoggedIn(
									A2($author$project$Main$enterPage, page, state))
							}),
						A2($author$project$Main$routeLoadCmd, state.accessToken, page));
				} else {
					return _Utils_Tuple2(
						_Utils_update(
							model,
							{route: page}),
						$elm$core$Platform$Cmd$none);
				}
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
var $author$project$Main$EmailChanged = function (a) {
	return {$: 'EmailChanged', a: a};
};
var $author$project$Main$LoginClicked = {$: 'LoginClicked'};
var $author$project$Main$PasswordChanged = function (a) {
	return {$: 'PasswordChanged', a: a};
};
var $author$project$Main$RegisterClicked = {$: 'RegisterClicked'};
var $elm$html$Html$form = _VirtualDom_node('form');
var $elm$html$Html$p = _VirtualDom_node('p');
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
var $author$project$Main$maybeError = F2(
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
var $author$project$Sharecrop$Ui$primaryButtonClass = 'rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50';
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
var $author$project$Sharecrop$Ui$secondaryButtonClass = 'rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100';
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
var $author$project$Sharecrop$Ui$fieldClass = 'w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none';
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
var $author$project$Main$authView = function (model) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Main$LoginClicked)
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
						$elm$html$Html$Events$onInput($author$project$Main$EmailChanged),
						$author$project$Sharecrop$Ui$testId('email')
					])),
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('password'),
						$elm$html$Html$Attributes$placeholder('Password'),
						$elm$html$Html$Attributes$value(model.password),
						$elm$html$Html$Events$onInput($author$project$Main$PasswordChanged),
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
								$elm$html$Html$Events$onClick($author$project$Main$RegisterClicked),
								$author$project$Sharecrop$Ui$testId('register')
							]),
						'Register')
					])),
				A2($author$project$Main$maybeError, model.authError, 'auth-error')
			]));
};
var $author$project$Main$LogoutClicked = {$: 'LogoutClicked'};
var $elm$html$Html$a = _VirtualDom_node('a');
var $elm$html$Html$Attributes$href = function (url) {
	return A2(
		$elm$html$Html$Attributes$stringProperty,
		'href',
		_VirtualDom_noJavaScriptUri(url));
};
var $author$project$Main$pageToPath = function (page) {
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
		case 'SeriesDetailPage':
			var seriesId = page.a;
			return '/series/' + seriesId;
		default:
			var teamId = page.a;
			return '/teams/' + teamId;
	}
};
var $author$project$Main$navLink = F4(
	function (current, target, identifier, labelText) {
		var styleClass = _Utils_eq(
			$author$project$Main$pageToPath(current),
			$author$project$Main$pageToPath(target)) ? $author$project$Sharecrop$Ui$primaryButtonClass : $author$project$Sharecrop$Ui$secondaryButtonClass;
		return A2(
			$elm$html$Html$a,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$href(
					$author$project$Main$pageToPath(target)),
					$elm$html$Html$Attributes$class(styleClass),
					$author$project$Sharecrop$Ui$testId('nav-' + identifier)
				]),
			_List_fromArray(
				[
					$elm$html$Html$text(labelText)
				]));
	});
var $author$project$Main$navBar = function (current) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex flex-wrap items-center gap-2')
			]),
		_List_fromArray(
			[
				A4($author$project$Main$navLink, current, $author$project$Main$OverviewPage, 'overview', 'Overview'),
				A4($author$project$Main$navLink, current, $author$project$Main$TasksPage, 'tasks', 'Tasks'),
				A4($author$project$Main$navLink, current, $author$project$Main$CreateTaskPage, 'create-task', 'New task'),
				A4($author$project$Main$navLink, current, $author$project$Main$DiscoveryPage, 'discovery', 'Discovery'),
				A4($author$project$Main$navLink, current, $author$project$Main$FundingPage, 'funding', 'Funding'),
				A4($author$project$Main$navLink, current, $author$project$Main$AgentsPage, 'agents', 'Agents'),
				A4($author$project$Main$navLink, current, $author$project$Main$CollectiblesPage, 'collectibles', 'Collectibles'),
				A4($author$project$Main$navLink, current, $author$project$Main$OrganizationsPage, 'organizations', 'Organizations'),
				A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick($author$project$Main$LogoutClicked),
						$author$project$Sharecrop$Ui$testId('logout')
					]),
				'Log out')
			]));
};
var $author$project$Main$AgentLabelChanged = function (a) {
	return {$: 'AgentLabelChanged', a: a};
};
var $author$project$Main$CreateAgentClicked = {$: 'CreateAgentClicked'};
var $author$project$Main$allScopes = _List_fromArray(
	[$author$project$Sharecrop$Generated$Agent$AgentScopeTasksRead, $author$project$Sharecrop$Generated$Agent$AgentScopeTasksWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsWrite, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsRead, $author$project$Sharecrop$Generated$Agent$AgentScopeSubmissionsReview]);
var $author$project$Sharecrop$Ui$card = function (children) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm')
			]),
		children);
};
var $author$project$Main$credentialStateLabel = function (state) {
	if (state.$ === 'AgentCredentialStateActive') {
		return 'active';
	} else {
		return 'revoked';
	}
};
var $author$project$Main$RevokeClicked = function (a) {
	return {$: 'RevokeClicked', a: a};
};
var $elm$html$Html$span = _VirtualDom_node('span');
var $author$project$Main$revokeButton = function (credential) {
	var _v0 = credential.state;
	if (_v0.$ === 'AgentCredentialStateActive') {
		return A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Events$onClick(
					$author$project$Main$RevokeClicked(credential.id)),
					$author$project$Sharecrop$Ui$testId('revoke-credential')
				]),
			'Revoke');
	} else {
		return A2(
			$elm$html$Html$span,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('text-xs text-slate-400')
				]),
			_List_fromArray(
				[
					$elm$html$Html$text('revoked')
				]));
	}
};
var $author$project$Main$scopeTag = function (scope) {
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
var $author$project$Main$credentialRow = function (credential) {
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
								$author$project$Main$credentialStateLabel(credential.state) + (' · ' + A2(
									$elm$core$String$join,
									', ',
									A2($elm$core$List$map, $author$project$Main$scopeTag, credential.scopes))))
							]))
					])),
				$author$project$Main$revokeButton(credential)
			]));
};
var $author$project$Main$credentialsList = function (credentials) {
	return $elm$core$List$isEmpty(credentials) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('credentials')
			]),
		A2($elm$core$List$map, $author$project$Main$credentialRow, credentials));
};
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
var $author$project$Main$maybeNote = F2(
	function (message, identifier) {
		if (message.$ === 'Just') {
			var value = message.a;
			return A2($author$project$Sharecrop$Ui$noteText, identifier, value);
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Sharecrop$Ui$codeBlockClass = 'overflow-x-auto rounded-md bg-slate-900 p-3 text-xs text-slate-100';
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
var $author$project$Main$mcpConfig = F2(
	function (origin, secret) {
		return '{\n  \"mcpServers\": {\n    \"sharecrop\": {\n      \"url\": \"' + (origin + ('/mcp\",\n      \"headers\": { \"Authorization\": \"Bearer ' + (secret + '\" }\n    }\n  }\n}')));
	});
var $author$project$Main$newCredentialView = F2(
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
						A2($author$project$Main$mcpConfig, origin, credential.secret))
					]));
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Main$ToggleScope = function (a) {
	return {$: 'ToggleScope', a: a};
};
var $elm$html$Html$Attributes$boolProperty = F2(
	function (key, bool) {
		return A2(
			_VirtualDom_property,
			key,
			$elm$json$Json$Encode$bool(bool));
	});
var $elm$html$Html$Attributes$checked = $elm$html$Html$Attributes$boolProperty('checked');
var $elm$html$Html$label = _VirtualDom_node('label');
var $author$project$Main$scopeCheckbox = F2(
	function (selected, scope) {
		return A2(
			$elm$html$Html$label,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('flex items-center gap-2 text-sm')
				]),
			_List_fromArray(
				[
					A2(
					$elm$html$Html$input,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('checkbox'),
							$elm$html$Html$Attributes$checked(
							A2($elm$core$List$member, scope, selected)),
							$elm$html$Html$Events$onClick(
							$author$project$Main$ToggleScope(scope)),
							$author$project$Sharecrop$Ui$testId(
							'scope-' + $author$project$Main$scopeTag(scope))
						]),
					_List_Nil),
					A2(
					$elm$html$Html$span,
					_List_Nil,
					_List_fromArray(
						[
							$elm$html$Html$text(
							$author$project$Main$scopeTag(scope))
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
var $author$project$Main$agentsView = F2(
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
							$elm$html$Html$Events$onSubmit($author$project$Main$CreateAgentClicked)
						]),
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$textInput(
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('text'),
									$elm$html$Html$Attributes$placeholder('Agent label'),
									$elm$html$Html$Attributes$value(state.agentLabel),
									$elm$html$Html$Events$onInput($author$project$Main$AgentLabelChanged),
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
								$author$project$Main$scopeCheckbox(state.agentScopes),
								$author$project$Main$allScopes)),
							A2(
							$author$project$Sharecrop$Ui$primaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('submit'),
									$author$project$Sharecrop$Ui$testId('create-agent')
								]),
							'Create credential'),
							A2($author$project$Main$maybeNote, state.agentMessage, 'agent-message')
						])),
					A2($author$project$Main$newCredentialView, origin, state.newCredential),
					$author$project$Main$credentialsList(state.credentials)
				]));
	});
var $author$project$Main$collectibleKindTag = function (kind) {
	switch (kind.$) {
		case 'CollectibleKindUnique':
			return 'unique';
		case 'CollectibleKindEdition':
			return 'edition';
		default:
			return 'badge';
	}
};
var $author$project$Main$collectibleKindLabel = $author$project$Main$collectibleKindTag;
var $author$project$Main$collectiblePolicyTag = function (policy) {
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
var $author$project$Main$collectiblePolicyLabel = $author$project$Main$collectiblePolicyTag;
var $author$project$Main$collectibleDetailView = F2(
	function (collectibleId, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('/collectibles'),
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
											'Kind: ' + $author$project$Main$collectibleKindLabel(collectible.kind))
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
											'State: ' + $author$project$Main$collectibleStateLabel(collectible.state))
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
											'Transfer policy: ' + $author$project$Main$collectiblePolicyLabel(collectible.transferPolicy))
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
									$elm$html$Html$text('This collectible is not in your holdings.')
								]));
					}
				}()
				]));
	});
var $author$project$Main$AwardTaskIdChanged = function (a) {
	return {$: 'AwardTaskIdChanged', a: a};
};
var $elm$html$Html$option = _VirtualDom_node('option');
var $elm$html$Html$select = _VirtualDom_node('select');
var $author$project$Main$collectibleCountLabel = function (count) {
	return (count === 1) ? '1 collectible' : ($elm$core$String$fromInt(count) + ' collectibles');
};
var $author$project$Main$rewardLabel = F3(
	function (kind, amount, collectibleCount) {
		switch (kind) {
			case 'credit':
				return (collectibleCount > 0) ? ($elm$core$String$fromInt(amount) + (' credits + ' + $author$project$Main$collectibleCountLabel(collectibleCount))) : ($elm$core$String$fromInt(amount) + ' credits');
			case 'collectible':
				return $author$project$Main$collectibleCountLabel(collectibleCount);
			case 'bundle':
				return $elm$core$String$fromInt(amount) + (' credits + ' + $author$project$Main$collectibleCountLabel(collectibleCount));
			default:
				return (collectibleCount > 0) ? $author$project$Main$collectibleCountLabel(collectibleCount) : 'no reward';
		}
	});
var $elm$html$Html$Attributes$selected = $elm$html$Html$Attributes$boolProperty('selected');
var $author$project$Main$taskStateLabel = function (state) {
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
var $author$project$Main$taskOption = F2(
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
					item.title + (' · ' + ($author$project$Main$taskStateLabel(item.state) + (' · ' + A3($author$project$Main$rewardLabel, item.rewardKind, item.rewardCreditAmount, item.rewardCollectibleCount)))))
				]));
	});
var $author$project$Main$taskPicker = F4(
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
				A2(
					$elm$html$Html$option,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$value('')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Select task')
						])),
				A2(
					$elm$core$List$map,
					$author$project$Main$taskOption(selectedTaskId),
					tasks)));
	});
var $author$project$Main$awardForm = function (state) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-4 space-y-3')
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$label_('Award to a task'),
				A4($author$project$Main$taskPicker, 'award-task-id', state.awardTaskId, $author$project$Main$AwardTaskIdChanged, state.tasks),
				A2($author$project$Main$maybeNote, state.awardMessage, 'award-message')
			]));
};
var $author$project$Main$AwardClicked = function (a) {
	return {$: 'AwardClicked', a: a};
};
var $author$project$Main$awardCollectibleButton = function (collectible) {
	var _v0 = collectible.state;
	if (_v0.$ === 'CollectibleStateMinted') {
		return A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(
					$author$project$Main$AwardClicked(collectible.id)),
					$author$project$Sharecrop$Ui$testId('award-collectible')
				]),
			'Award');
	} else {
		return $elm$html$Html$text('');
	}
};
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
var $author$project$Main$collectibleRow = function (collectible) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between py-2'),
				$author$project$Sharecrop$Ui$testId('collectible-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex items-center gap-2')
					]),
				_List_fromArray(
					[
						A2(
						$elm$html$Html$a,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$href('/collectibles/' + collectible.id),
								$elm$html$Html$Attributes$class('font-medium underline'),
								$author$project$Sharecrop$Ui$testId('collectible-link')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(collectible.name)
							])),
						$author$project$Sharecrop$Ui$badge(
						$author$project$Main$collectibleStateLabel(collectible.state)),
						A2(
						$elm$html$Html$span,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-xs text-slate-500')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								$author$project$Main$collectibleKindLabel(collectible.kind))
							]))
					])),
				$author$project$Main$awardCollectibleButton(collectible)
			]));
};
var $author$project$Main$collectiblesList = function (state) {
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
		A2($elm$core$List$map, $author$project$Main$collectibleRow, state.collectibles));
};
var $author$project$Main$CollectibleNameChanged = function (a) {
	return {$: 'CollectibleNameChanged', a: a};
};
var $author$project$Main$MintClicked = {$: 'MintClicked'};
var $author$project$Main$allKinds = _List_fromArray(
	[$author$project$Sharecrop$Generated$Collectible$CollectibleKindUnique, $author$project$Sharecrop$Generated$Collectible$CollectibleKindEdition, $author$project$Sharecrop$Generated$Collectible$CollectibleKindBadge]);
var $author$project$Main$allPolicies = _List_fromArray(
	[$author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyNonTransferableExceptPayout, $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableBetweenUsers, $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyTransferableWithinOrganization, $author$project$Sharecrop$Generated$Collectible$CollectibleTransferPolicyIssuerControlled]);
var $author$project$Main$CollectibleKindChosen = function (a) {
	return {$: 'CollectibleKindChosen', a: a};
};
var $author$project$Main$chooserButton = F4(
	function (isSelected, msg, identifier, labelText) {
		return isSelected ? A2(
			$author$project$Sharecrop$Ui$primaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(msg),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			labelText) : A2(
			$author$project$Sharecrop$Ui$secondaryButton,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$type_('button'),
					$elm$html$Html$Events$onClick(msg),
					$author$project$Sharecrop$Ui$testId(identifier)
				]),
			labelText);
	});
var $author$project$Main$kindButton = F2(
	function (selected, kind) {
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(selected, kind),
			$author$project$Main$CollectibleKindChosen(kind),
			'collectible-kind-' + $author$project$Main$collectibleKindTag(kind),
			$author$project$Main$collectibleKindLabel(kind));
	});
var $author$project$Main$CollectiblePolicyChosen = function (a) {
	return {$: 'CollectiblePolicyChosen', a: a};
};
var $author$project$Main$policyButton = F2(
	function (selected, policy) {
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(selected, policy),
			$author$project$Main$CollectiblePolicyChosen(policy),
			'collectible-policy-' + $author$project$Main$collectiblePolicyTag(policy),
			$author$project$Main$collectiblePolicyLabel(policy));
	});
var $author$project$Main$mintForm = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('mt-3 space-y-3'),
				$elm$html$Html$Events$onSubmit($author$project$Main$MintClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('text'),
						$elm$html$Html$Attributes$placeholder('Collectible name'),
						$elm$html$Html$Attributes$value(state.collectibleName),
						$elm$html$Html$Events$onInput($author$project$Main$CollectibleNameChanged),
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
					$author$project$Main$kindButton(state.collectibleKind),
					$author$project$Main$allKinds)),
				$author$project$Sharecrop$Ui$label_('Transfer policy'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Main$policyButton(state.collectiblePolicy),
					$author$project$Main$allPolicies)),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$author$project$Sharecrop$Ui$testId('mint-collectible')
					]),
				'Mint collectible'),
				A2($author$project$Main$maybeNote, state.collectibleMessage, 'collectible-message')
			]));
};
var $author$project$Main$collectiblesView = function (state) {
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
						$elm$html$Html$text('Mint collectibles and award them to tasks.')
					])),
				$author$project$Main$mintForm(state),
				$author$project$Main$awardForm(state),
				$author$project$Main$collectiblesList(state)
			]));
};
var $author$project$Main$CreateDescriptionChanged = function (a) {
	return {$: 'CreateDescriptionChanged', a: a};
};
var $author$project$Main$CreateReservationHoursChanged = function (a) {
	return {$: 'CreateReservationHoursChanged', a: a};
};
var $author$project$Main$CreateRewardAmountChanged = function (a) {
	return {$: 'CreateRewardAmountChanged', a: a};
};
var $author$project$Main$CreateTaskClicked = {$: 'CreateTaskClicked'};
var $author$project$Main$CreateTitleChanged = function (a) {
	return {$: 'CreateTitleChanged', a: a};
};
var $author$project$Main$allAssigneeScopes = _List_fromArray(
	[$author$project$Sharecrop$Generated$Task$TaskAssigneeScopeUser, $author$project$Sharecrop$Generated$Task$TaskAssigneeScopeOrganizationTeam]);
var $author$project$Main$allParticipationPolicies = _List_fromArray(
	[$author$project$Sharecrop$Generated$Task$TaskParticipationPolicyOpen, $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyReservationRequired, $author$project$Sharecrop$Generated$Task$TaskParticipationPolicyApprovalRequired]);
var $author$project$Main$visibilityPublicTag = 'public';
var $author$project$Main$allVisibilityTags = _List_fromArray(
	[$author$project$Main$visibilityPublicTag, $author$project$Main$visibilityDefaultTag, $author$project$Main$visibilityUserTag, $author$project$Main$visibilityTeamTag, $author$project$Main$visibilityOrganizationTag]);
var $author$project$Main$CreateAssigneeScopeChosen = function (a) {
	return {$: 'CreateAssigneeScopeChosen', a: a};
};
var $author$project$Main$assigneeScopeLabel = function (scope) {
	if (scope.$ === 'TaskAssigneeScopeUser') {
		return 'user';
	} else {
		return 'organization team';
	}
};
var $author$project$Main$assigneeScopeButton = F2(
	function (selected, scope) {
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(selected, scope),
			$author$project$Main$CreateAssigneeScopeChosen(scope),
			'create-assignee-' + $author$project$Main$assigneeScopeTag(scope),
			$author$project$Main$assigneeScopeLabel(scope));
	});
var $author$project$Sharecrop$Ui$fieldLabel = F2(
	function (labelText, controls) {
		return A2(
			$elm$html$Html$label,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('block space-y-1 text-sm font-medium text-slate-700')
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
var $author$project$Main$CreateTaskOwnerChanged = function (a) {
	return {$: 'CreateTaskOwnerChanged', a: a};
};
var $author$project$Main$ownerButton = F2(
	function (selected, organization) {
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(selected, organization.id),
			$author$project$Main$CreateTaskOwnerChanged(organization.id),
			'create-owner-' + organization.id,
			organization.name);
	});
var $author$project$Main$ownerChooser = function (state) {
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
						$author$project$Main$chooserButton,
						state.createTaskOwner === '',
						$author$project$Main$CreateTaskOwnerChanged(''),
						'create-owner-me',
						'Me'),
					A2(
						$elm$core$List$map,
						$author$project$Main$ownerButton(state.createTaskOwner),
						state.organizations)))
			]));
};
var $author$project$Main$CreateParticipationChanged = function (a) {
	return {$: 'CreateParticipationChanged', a: a};
};
var $author$project$Main$participationPolicyLabel = function (policy) {
	switch (policy.$) {
		case 'TaskParticipationPolicyOpen':
			return 'open submissions';
		case 'TaskParticipationPolicyReservationRequired':
			return 'reservation required';
		default:
			return 'approval required';
	}
};
var $author$project$Main$participationButton = F2(
	function (selectedPolicy, policy) {
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(
				selectedPolicy,
				$author$project$Main$participationPolicyTag(policy)),
			$author$project$Main$CreateParticipationChanged(
				$author$project$Main$participationPolicyTag(policy)),
			'create-participation-' + $author$project$Main$participationPolicyTag(policy),
			$author$project$Main$participationPolicyLabel(policy));
	});
var $elm$html$Html$Attributes$rows = function (n) {
	return A2(
		_VirtualDom_attribute,
		'rows',
		$elm$core$String$fromInt(n));
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
var $author$project$Main$CreateVisibilityChanged = function (a) {
	return {$: 'CreateVisibilityChanged', a: a};
};
var $author$project$Main$visibilityLabel = function (tag) {
	return _Utils_eq(tag, $author$project$Main$visibilityPublicTag) ? 'Public' : (_Utils_eq(tag, $author$project$Main$visibilityUserTag) ? 'Specific user' : (_Utils_eq(tag, $author$project$Main$visibilityTeamTag) ? 'Team' : (_Utils_eq(tag, $author$project$Main$visibilityOrganizationTag) ? 'Organization' : 'Private (default)')));
};
var $author$project$Main$visibilityButton = F2(
	function (selected, tag) {
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(selected, tag),
			$author$project$Main$CreateVisibilityChanged(tag),
			'create-visibility-' + tag,
			$author$project$Main$visibilityLabel(tag));
	});
var $author$project$Main$CreateScopeOrganizationIdChanged = function (a) {
	return {$: 'CreateScopeOrganizationIdChanged', a: a};
};
var $author$project$Main$CreateScopeTeamIdChanged = function (a) {
	return {$: 'CreateScopeTeamIdChanged', a: a};
};
var $author$project$Main$CreateScopeUserIdChanged = function (a) {
	return {$: 'CreateScopeUserIdChanged', a: a};
};
var $author$project$Main$visibilityScopeField = function (state) {
	return _Utils_eq(state.createVisibility, $author$project$Main$visibilityUserTag) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Share with user ID',
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('text'),
						$elm$html$Html$Attributes$placeholder('User ID to grant access'),
						$elm$html$Html$Attributes$value(state.createScopeUserId),
						$elm$html$Html$Events$onInput($author$project$Main$CreateScopeUserIdChanged),
						$author$project$Sharecrop$Ui$testId('create-scope-user')
					]))
			])) : (_Utils_eq(state.createVisibility, $author$project$Main$visibilityTeamTag) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Share with team ID',
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('text'),
						$elm$html$Html$Attributes$placeholder('Team ID (standalone or organization team)'),
						$elm$html$Html$Attributes$value(state.createScopeTeamId),
						$elm$html$Html$Events$onInput($author$project$Main$CreateScopeTeamIdChanged),
						$author$project$Sharecrop$Ui$testId('create-scope-team')
					]))
			])) : (_Utils_eq(state.createVisibility, $author$project$Main$visibilityOrganizationTag) ? A2(
		$author$project$Sharecrop$Ui$fieldLabel,
		'Share with organization ID',
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('text'),
						$elm$html$Html$Attributes$placeholder('Organization ID'),
						$elm$html$Html$Attributes$value(state.createScopeOrganizationId),
						$elm$html$Html$Events$onInput($author$project$Main$CreateScopeOrganizationIdChanged),
						$author$project$Sharecrop$Ui$testId('create-scope-organization')
					]))
			])) : $elm$html$Html$text('')));
};
var $author$project$Main$createTaskView = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Main$CreateTaskClicked)
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
								$elm$html$Html$Events$onInput($author$project$Main$CreateTitleChanged),
								$author$project$Sharecrop$Ui$testId('create-title')
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
								$elm$html$Html$Attributes$placeholder('What the worker should do'),
								$elm$html$Html$Attributes$value(state.createDescription),
								$elm$html$Html$Events$onInput($author$project$Main$CreateDescriptionChanged),
								$elm$html$Html$Attributes$rows(3),
								$author$project$Sharecrop$Ui$testId('create-description')
							]))
					])),
				A2(
				$author$project$Sharecrop$Ui$fieldLabel,
				'Credit reward',
				_List_fromArray(
					[
						$author$project$Sharecrop$Ui$textInput(
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('number'),
								$elm$html$Html$Attributes$placeholder('Blank for no reward'),
								$elm$html$Html$Attributes$value(state.createRewardAmount),
								$elm$html$Html$Events$onInput($author$project$Main$CreateRewardAmountChanged),
								$author$project$Sharecrop$Ui$testId('create-reward')
							]))
					])),
				$author$project$Main$ownerChooser(state),
				$author$project$Sharecrop$Ui$label_('Participation'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Main$participationButton(state.createParticipationPolicy),
					$author$project$Main$allParticipationPolicies)),
				A2(
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
								$elm$html$Html$Events$onInput($author$project$Main$CreateReservationHoursChanged),
								$author$project$Sharecrop$Ui$testId('create-reservation-hours')
							]))
					])),
				$author$project$Sharecrop$Ui$label_('Visibility'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Main$visibilityButton(state.createVisibility),
					$author$project$Main$allVisibilityTags)),
				$author$project$Main$visibilityScopeField(state),
				$author$project$Sharecrop$Ui$label_('Assignee'),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				A2(
					$elm$core$List$map,
					$author$project$Main$assigneeScopeButton(state.createAssigneeScope),
					$author$project$Main$allAssigneeScopes)),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$author$project$Sharecrop$Ui$testId('create-task')
					]),
				'Create task'),
				A2($author$project$Main$maybeNote, state.createMessage, 'create-message')
			]));
};
var $author$project$Main$DiscoveryIncludeReservedChanged = function (a) {
	return {$: 'DiscoveryIncludeReservedChanged', a: a};
};
var $author$project$Sharecrop$Ui$checkboxClass = 'h-4 w-4 rounded border-slate-400 text-slate-900 focus:ring-2 focus:ring-slate-500';
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
var $author$project$Main$DiscoveryViewClicked = function (a) {
	return {$: 'DiscoveryViewClicked', a: a};
};
var $author$project$Main$activeAssigneeSuffix = function (item) {
	return (item.activeAssigneeID === '') ? '' : (' · reserved by ' + item.activeAssigneeID);
};
var $author$project$Main$discoveryRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between py-2'),
				$author$project$Sharecrop$Ui$testId('discovery-task-row')
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
								$elm$html$Html$text(item.title)
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
								$author$project$Main$taskStateLabel(item.state) + (' · ' + (A3($author$project$Main$rewardLabel, item.rewardKind, item.rewardCreditAmount, item.rewardCollectibleCount) + (' · ' + ($author$project$Main$participationPolicyLabel(item.participationPolicy) + $author$project$Main$activeAssigneeSuffix(item))))))
							]))
					])),
				A2(
				$author$project$Sharecrop$Ui$secondaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Events$onClick(
						$author$project$Main$DiscoveryViewClicked(item.id)),
						$author$project$Sharecrop$Ui$testId('discovery-view')
					]),
				'View')
			]));
};
var $author$project$Main$discoveryList = function (tasks) {
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
		A2($elm$core$List$map, $author$project$Main$discoveryRow, tasks));
};
var $elm$core$Basics$not = _Basics_not;
var $author$project$Main$discoveryView = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Discover public tasks'),
				A2(
				$author$project$Sharecrop$Ui$checkbox,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$checked(state.discoveryIncludeReserved),
						$elm$html$Html$Events$onClick(
						$author$project$Main$DiscoveryIncludeReservedChanged(!state.discoveryIncludeReserved)),
						$author$project$Sharecrop$Ui$testId('include-reserved')
					]),
				'Include reserved'),
				$author$project$Main$discoveryList(state.discoveryTasks)
			]));
};
var $author$project$Main$FundAmountChanged = function (a) {
	return {$: 'FundAmountChanged', a: a};
};
var $author$project$Main$FundClicked = {$: 'FundClicked'};
var $author$project$Main$FundOrganizationIdChanged = function (a) {
	return {$: 'FundOrganizationIdChanged', a: a};
};
var $author$project$Main$FundTaskIdChanged = function (a) {
	return {$: 'FundTaskIdChanged', a: a};
};
var $elm$html$Html$Attributes$disabled = $elm$html$Html$Attributes$boolProperty('disabled');
var $author$project$Main$fundingView = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Main$FundClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Fund a task'),
				A4($author$project$Main$taskPicker, 'fund-task-id', state.fundTaskId, $author$project$Main$FundTaskIdChanged, state.tasks),
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('number'),
						$elm$html$Html$Attributes$placeholder('Amount in credits'),
						$elm$html$Html$Attributes$value(state.fundAmount),
						$elm$html$Html$Events$onInput($author$project$Main$FundAmountChanged),
						$author$project$Sharecrop$Ui$testId('fund-amount')
					])),
				$author$project$Sharecrop$Ui$textInput(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('text'),
						$elm$html$Html$Attributes$placeholder('Organization ID (optional — fund from org credits)'),
						$elm$html$Html$Attributes$value(state.fundOrganizationId),
						$elm$html$Html$Events$onInput($author$project$Main$FundOrganizationIdChanged),
						$author$project$Sharecrop$Ui$testId('fund-organization')
					])),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$elm$html$Html$Attributes$disabled(state.fundTaskId === ''),
						$author$project$Sharecrop$Ui$testId('fund')
					]),
				'Fund task'),
				A2($author$project$Main$maybeNote, state.fundMessage, 'fund-message')
			]));
};
var $author$project$Main$CreateOrgTeamClicked = {$: 'CreateOrgTeamClicked'};
var $author$project$Main$CreateOrgTeamNameChanged = function (a) {
	return {$: 'CreateOrgTeamNameChanged', a: a};
};
var $author$project$Main$ProvisionMemberClicked = {$: 'ProvisionMemberClicked'};
var $author$project$Main$ProvisionMemberEmailChanged = function (a) {
	return {$: 'ProvisionMemberEmailChanged', a: a};
};
var $author$project$Main$balanceLabel = function (balance) {
	if (balance.$ === 'Just') {
		var amount = balance.a;
		return $elm$core$String$fromInt(amount) + ' credits';
	} else {
		return 'Loading…';
	}
};
var $author$project$Main$membershipStatusText = function (status) {
	switch (status.$) {
		case 'MembershipStatusActive':
			return 'active';
		case 'MembershipStatusDeactivated':
			return 'deactivated';
		default:
			return 'removed';
	}
};
var $author$project$Main$organizationRoleText = function (role) {
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
var $author$project$Main$orgMemberRow = function (member) {
	var roles = $elm$core$List$isEmpty(member.roles) ? 'no roles' : A2(
		$elm$core$String$join,
		', ',
		A2($elm$core$List$map, $author$project$Main$organizationRoleText, member.roles));
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between gap-2 py-2'),
				$author$project$Sharecrop$Ui$testId('org-member-row')
			]),
		_List_fromArray(
			[
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('/users/' + member.userID),
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
						roles + (' · ' + $author$project$Main$membershipStatusText(member.status)))
					]))
			]));
};
var $author$project$Main$orgMembersList = function (members) {
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
		A2($elm$core$List$map, $author$project$Main$orgMemberRow, members));
};
var $author$project$Main$orgTeamsList = function (teams) {
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
							$elm$html$Html$Attributes$href('/teams/' + team.id),
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
var $author$project$Main$tasksListSimple = F2(
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
								item.title + (' · ' + $author$project$Main$taskStateLabel(item.state)))
							]));
				},
				tasks));
	});
var $author$project$Main$activeOrganizationView = function (state) {
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
				'Balance: ' + $author$project$Main$balanceLabel(state.orgBalance)),
				$author$project$Sharecrop$Ui$sectionTitle('Organization tasks'),
				A2($author$project$Main$tasksListSimple, 'org-tasks', state.orgTasks),
				$author$project$Sharecrop$Ui$sectionTitle('Teams'),
				$author$project$Main$orgTeamsList(state.orgTeams),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit($author$project$Main$CreateOrgTeamClicked)
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
										$elm$html$Html$Events$onInput($author$project$Main$CreateOrgTeamNameChanged),
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
				A2($author$project$Main$maybeNote, state.orgTeamMessage, 'org-team-message'),
				$author$project$Sharecrop$Ui$sectionTitle('Members'),
				$author$project$Main$orgMembersList(state.orgMembers),
				$author$project$Sharecrop$Ui$sectionTitle('Provision a member'),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit($author$project$Main$ProvisionMemberClicked)
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
										$elm$html$Html$Events$onInput($author$project$Main$ProvisionMemberEmailChanged),
										$author$project$Sharecrop$Ui$testId('provision-member-email')
									]))
							])),
						A2(
						$author$project$Sharecrop$Ui$primaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$type_('submit'),
								$author$project$Sharecrop$Ui$testId('provision-member')
							]),
						'Provision member')
					])),
				A2($author$project$Main$maybeNote, state.provisionMemberMessage, 'provision-member-message')
			]));
};
var $elm$core$List$head = function (list) {
	if (list.b) {
		var x = list.a;
		var xs = list.b;
		return $elm$core$Maybe$Just(x);
	} else {
		return $elm$core$Maybe$Nothing;
	}
};
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
var $author$project$Main$organizationDetailView = function (state) {
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
						$elm$html$Html$Attributes$href('/organizations'),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
						$author$project$Sharecrop$Ui$testId('back-organizations')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Back to organizations')
					])),
				$author$project$Sharecrop$Ui$sectionTitle(name),
				$author$project$Main$activeOrganizationView(state)
			]));
};
var $author$project$Main$CreateOrgClicked = {$: 'CreateOrgClicked'};
var $author$project$Main$CreateOrgNameChanged = function (a) {
	return {$: 'CreateOrgNameChanged', a: a};
};
var $author$project$Main$organizationRow = function (organization) {
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
						$elm$html$Html$Attributes$href('/organizations/' + organization.id),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
						$author$project$Sharecrop$Ui$testId('select-organization')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('Open')
					]))
			]));
};
var $author$project$Main$organizationsList = function (state) {
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
		A2($elm$core$List$map, $author$project$Main$organizationRow, state.organizations));
};
var $author$project$Main$organizationsView = function (state) {
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
				$author$project$Main$organizationsList(state),
				A2(
				$elm$html$Html$form,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('mt-3 flex flex-wrap items-end gap-2'),
						$elm$html$Html$Events$onSubmit($author$project$Main$CreateOrgClicked)
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
										$elm$html$Html$Events$onInput($author$project$Main$CreateOrgNameChanged),
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
				A2($author$project$Main$maybeNote, state.orgMessage, 'org-message')
			]));
};
var $author$project$Main$balanceView = function (balance) {
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
						$author$project$Main$balanceLabel(balance))
					]))
			]));
};
var $author$project$Main$kindLabel = function (kind) {
	switch (kind.$) {
		case 'LedgerEntryKindSignupGrant':
			return 'signup_grant';
		case 'LedgerEntryKindTaskEscrow':
			return 'task_escrow';
		case 'LedgerEntryKindTaskRefund':
			return 'task_refund';
		case 'LedgerEntryKindTaskPayout':
			return 'task_payout';
		case 'LedgerEntryKindTaskTip':
			return 'task_tip';
		default:
			return 'manual_adjustment';
	}
};
var $elm$html$Html$td = _VirtualDom_node('td');
var $elm$html$Html$tr = _VirtualDom_node('tr');
var $author$project$Main$ledgerRow = function (entry) {
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
						$author$project$Main$kindLabel(entry.kind))
					])),
				A2(
				$elm$html$Html$td,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('py-2 text-right tabular-nums')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text(
						$elm$core$String$fromInt(entry.amount))
					]))
			]));
};
var $elm$html$Html$table = _VirtualDom_node('table');
var $elm$html$Html$tbody = _VirtualDom_node('tbody');
var $elm$html$Html$th = _VirtualDom_node('th');
var $elm$html$Html$thead = _VirtualDom_node('thead');
var $author$project$Main$ledgerView = function (entries) {
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
						A2($elm$core$List$map, $author$project$Main$ledgerRow, entries))
					]))
			]));
};
var $author$project$Main$overviewView = function (state) {
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
				$author$project$Main$balanceView(state.balance),
				$author$project$Main$ledgerView(state.entries)
			]));
};
var $author$project$Main$seriesDetailView = F2(
	function (seriesId, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					function () {
					var _v0 = state.seriesDetail;
					if (_v0.$ === 'Just') {
						var series = _v0.a;
						return A2(
							$elm$html$Html$div,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('space-y-2'),
									$author$project$Sharecrop$Ui$testId('series-detail')
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
											$elm$html$Html$text(series.title)
										])),
									$author$project$Sharecrop$Ui$label_('Series ' + series.id),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('Owner kind: ' + series.ownerKind)
										])),
									A2(
									$elm$html$Html$p,
									_List_fromArray(
										[
											$elm$html$Html$Attributes$class('text-sm')
										]),
									_List_fromArray(
										[
											$elm$html$Html$text('Created by: ' + series.createdBy)
										]))
								]));
					} else {
						return A2(
							$elm$html$Html$p,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$class('text-sm text-slate-500'),
									$author$project$Sharecrop$Ui$testId('series-detail-missing')
								]),
							_List_fromArray(
								[
									$elm$html$Html$text('Loading series ' + (seriesId + '…'))
								]));
					}
				}()
				]));
	});
var $author$project$Main$availabilityKindLabel = function (kind) {
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
var $author$project$Main$mcpInitializeCurl = function (origin) {
	return 'curl -i -X POST ' + (origin + '/mcp \\\n  -H \"Authorization: Bearer <AGENT_TOKEN>\" \\\n  -H \"Accept: application/json, text/event-stream\" \\\n  -H \"Content-Type: application/json\" \\\n  -d \'{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{}}\'');
};
var $author$project$Main$mcpSchemaCurl = F2(
	function (origin, taskId) {
		return 'curl -X POST ' + (origin + ('/mcp \\\n  -H \"Authorization: Bearer <AGENT_TOKEN>\" \\\n  -H \"Mcp-Session-Id: <MCP_SESSION_ID>\" \\\n  -H \"Accept: application/json, text/event-stream\" \\\n  -H \"Content-Type: application/json\" \\\n  -d \'{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.get_task_schema\",\"arguments\":{\"task_id\":\"' + (taskId + '\"}}}\'')));
	});
var $author$project$Main$mcpSubmitCurl = F2(
	function (origin, taskId) {
		return 'curl -X POST ' + (origin + ('/mcp \\\n  -H \"Authorization: Bearer <AGENT_TOKEN>\" \\\n  -H \"Mcp-Session-Id: <MCP_SESSION_ID>\" \\\n  -H \"Accept: application/json, text/event-stream\" \\\n  -H \"Content-Type: application/json\" \\\n  -d \'{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"sharecrop.submit_response\",\"arguments\":{\"task_id\":\"' + (taskId + '\",\"response_json\":\"{}\"}}}\'')));
	});
var $author$project$Main$restReserveCurl = F2(
	function (origin, taskId) {
		return 'curl -X POST ' + (origin + ('/api/tasks/' + (taskId + '/reservations \\\n  -H \"Authorization: Bearer <ACCESS_TOKEN>\"')));
	});
var $author$project$Main$restSubmitCurl = F2(
	function (origin, taskId) {
		return 'curl -X POST ' + (origin + ('/api/tasks/' + (taskId + '/submissions \\\n  -H \"Authorization: Bearer <ACCESS_TOKEN>\" \\\n  -H \"Content-Type: application/json\" \\\n  -d \'{\"response_json\":\"{}\"}\'')));
	});
var $author$project$Main$taskInstructions = F2(
	function (origin, taskId) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-3'),
					$author$project$Sharecrop$Ui$testId('task-instructions')
				]),
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$label_('REST API'),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('task-rest-submit')
						]),
					A2($author$project$Main$restSubmitCurl, origin, taskId)),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('task-rest-reserve')
						]),
					A2($author$project$Main$restReserveCurl, origin, taskId)),
					$author$project$Sharecrop$Ui$label_('MCP'),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('task-mcp-initialize')
						]),
					$author$project$Main$mcpInitializeCurl(origin)),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('task-mcp-submit')
						]),
					A2($author$project$Main$mcpSubmitCurl, origin, taskId)),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('task-mcp-schema')
						]),
					A2($author$project$Main$mcpSchemaCurl, origin, taskId))
				]));
	});
var $author$project$Main$detailCard = F2(
	function (origin, state) {
		var _v0 = state.detail;
		if (_v0.$ === 'Just') {
			var detail = _v0.a;
			return $author$project$Sharecrop$Ui$card(
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
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$badge(
								$author$project$Main$taskStateLabel(detail.state)),
								$author$project$Sharecrop$Ui$badge(
								$author$project$Main$availabilityKindLabel(detail.availabilityKind)),
								$author$project$Sharecrop$Ui$badge(
								$author$project$Main$participationPolicyLabel(detail.participationPolicy))
							])),
						A2(
						$elm$html$Html$p,
						_List_fromArray(
							[
								$elm$html$Html$Attributes$class('text-sm font-medium')
							]),
						_List_fromArray(
							[
								$elm$html$Html$text(
								'Reward: ' + A3($author$project$Main$rewardLabel, detail.rewardKind, detail.rewardCreditAmount, detail.rewardCollectibleCount))
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
							])),
						$author$project$Sharecrop$Ui$label_('Response schema'),
						A2(
						$author$project$Sharecrop$Ui$codeBlock,
						_List_fromArray(
							[
								$author$project$Sharecrop$Ui$testId('detail-schema')
							]),
						detail.responseSchemaJson),
						A2($author$project$Main$taskInstructions, origin, detail.id)
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
	});
var $author$project$Main$OpenTaskClicked = function (a) {
	return {$: 'OpenTaskClicked', a: a};
};
var $author$project$Main$RefundTaskClicked = function (a) {
	return {$: 'RefundTaskClicked', a: a};
};
var $author$project$Main$taskStateGuidance = function (state) {
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
var $author$project$Main$ownerControlsCard = function (state) {
	var _v0 = state.detail;
	if (_v0.$ === 'Just') {
		var detail = _v0.a;
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
							$author$project$Main$taskStateGuidance(detail.state))
						])),
					A2(
					$elm$html$Html$div,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$class('flex gap-2')
						]),
					_List_fromArray(
						[
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									$author$project$Main$OpenTaskClicked(detail.id)),
									$author$project$Sharecrop$Ui$testId('open-task')
								]),
							'Open'),
							A2(
							$author$project$Sharecrop$Ui$secondaryButton,
							_List_fromArray(
								[
									$elm$html$Html$Attributes$type_('button'),
									$elm$html$Html$Events$onClick(
									$author$project$Main$RefundTaskClicked(detail.id)),
									$author$project$Sharecrop$Ui$testId('refund-task')
								]),
							'Refund')
						])),
					A2($author$project$Main$maybeNote, state.createMessage, 'create-message')
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Main$ReserveClicked = function (a) {
	return {$: 'ReserveClicked', a: a};
};
var $author$project$Main$reservationAction = function (detail) {
	var _v0 = detail.viewerAction;
	switch (_v0.$) {
		case 'TaskViewerActionReserve':
			return A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Main$ReserveClicked(detail.id)),
						$author$project$Sharecrop$Ui$testId('reserve-task')
					]),
				'Reserve');
		case 'TaskViewerActionRequestApproval':
			return A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('button'),
						$elm$html$Html$Events$onClick(
						$author$project$Main$ReserveClicked(detail.id)),
						$author$project$Sharecrop$Ui$testId('request-approval')
					]),
				'Request approval');
		default:
			return $elm$html$Html$text('');
	}
};
var $author$project$Main$ApproveReservationClicked = function (a) {
	return {$: 'ApproveReservationClicked', a: a};
};
var $author$project$Main$CancelReservationClicked = function (a) {
	return {$: 'CancelReservationClicked', a: a};
};
var $author$project$Main$DeclineReservationClicked = function (a) {
	return {$: 'DeclineReservationClicked', a: a};
};
var $author$project$Main$reservationButtons = function (reservation) {
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
							$author$project$Main$ApproveReservationClicked(reservation.id)),
							$author$project$Sharecrop$Ui$testId('approve-reservation')
						]),
					'Approve'),
					A2(
					$author$project$Sharecrop$Ui$secondaryButton,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$type_('button'),
							$elm$html$Html$Events$onClick(
							$author$project$Main$DeclineReservationClicked(reservation.id)),
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
							$author$project$Main$CancelReservationClicked(reservation.id)),
							$author$project$Sharecrop$Ui$testId('cancel-reservation')
						]),
					'Cancel')
				]);
		default:
			return _List_Nil;
	}
};
var $author$project$Main$reservationRow = function (reservation) {
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
								reservation.assigneeID + (' · ' + $author$project$Main$assigneeScopeLabel(reservation.assigneeKind)))
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
								$author$project$Main$reservationStateLabel(reservation.state))
							]))
					])),
				A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap gap-2')
					]),
				$author$project$Main$reservationButtons(reservation))
			]));
};
var $author$project$Main$reservationsList = function (reservations) {
	return $elm$core$List$isEmpty(reservations) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('divide-y divide-slate-100'),
				$author$project$Sharecrop$Ui$testId('reservations')
			]),
		A2($elm$core$List$map, $author$project$Main$reservationRow, reservations));
};
var $author$project$Main$viewerActionLabel = function (action) {
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
var $author$project$Main$reservationCard = function (state) {
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
							$author$project$Main$viewerActionLabel(detail.viewerAction)),
							$author$project$Sharecrop$Ui$badge(
							$author$project$Main$assigneeScopeLabel(detail.assigneeScope))
						])),
					$author$project$Main$reservationAction(detail),
					$author$project$Main$reservationsList(state.reservations),
					A2($author$project$Main$maybeNote, state.reservationMessage, 'reservation-message')
				]));
	} else {
		return $elm$html$Html$text('');
	}
};
var $author$project$Main$ReviewBanChanged = function (a) {
	return {$: 'ReviewBanChanged', a: a};
};
var $author$project$Main$ReviewNoteChanged = function (a) {
	return {$: 'ReviewNoteChanged', a: a};
};
var $author$project$Main$ReviewPartialCreditChanged = function (a) {
	return {$: 'ReviewPartialCreditChanged', a: a};
};
var $author$project$Main$ReviewTipChanged = function (a) {
	return {$: 'ReviewTipChanged', a: a};
};
var $elm$json$Json$Decode$bool = _Json_decodeBool;
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
var $author$project$Main$reviewControls = function (state) {
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
								$elm$html$Html$Events$onInput($author$project$Main$ReviewNoteChanged),
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
										$elm$html$Html$Events$onInput($author$project$Main$ReviewPartialCreditChanged),
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
										$elm$html$Html$Events$onInput($author$project$Main$ReviewTipChanged),
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
										$elm$html$Html$Events$onCheck($author$project$Main$ReviewBanChanged),
										$author$project$Sharecrop$Ui$testId('review-ban')
									]),
								'Ban implementor')
							]))
					])),
				A2($author$project$Main$maybeNote, state.reviewMessage, 'review-message')
			]));
};
var $author$project$Main$AcceptClicked = function (a) {
	return {$: 'AcceptClicked', a: a};
};
var $author$project$Main$RejectClicked = function (a) {
	return {$: 'RejectClicked', a: a};
};
var $author$project$Main$RequestChangesClicked = function (a) {
	return {$: 'RequestChangesClicked', a: a};
};
var $author$project$Main$reviewButtons = F2(
	function (state, submission) {
		var _v0 = submission.state;
		if (_v0.$ === 'SubmissionStateSubmitted') {
			return A2(
				$elm$html$Html$div,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('flex flex-wrap justify-end gap-2')
					]),
				_List_fromArray(
					[
						A2(
						$author$project$Sharecrop$Ui$secondaryButton,
						_List_fromArray(
							[
								$elm$html$Html$Events$onClick(
								$author$project$Main$RequestChangesClicked(submission.id)),
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
								$author$project$Main$RejectClicked(submission.id)),
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
								$author$project$Main$AcceptClicked(submission.id)),
								$author$project$Sharecrop$Ui$testId('accept-submission')
							]),
						'Accept')
					]));
		} else {
			return $elm$html$Html$text('');
		}
	});
var $author$project$Main$reviewNoteView = function (note) {
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
var $author$project$Main$validationErrorView = function (item) {
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
var $author$project$Main$validationErrorsView = function (errors) {
	return $elm$core$List$isEmpty(errors) ? $elm$html$Html$text('') : A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-1')
			]),
		A2($elm$core$List$map, $author$project$Main$validationErrorView, errors));
};
var $author$project$Main$submissionRow = F2(
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
							$author$project$Main$submissionStateLabel(submission.state)),
							A2($author$project$Main$reviewButtons, state, submission)
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
					$author$project$Main$reviewNoteView(submission.reviewNote),
					A2(
					$author$project$Sharecrop$Ui$codeBlock,
					_List_fromArray(
						[
							$author$project$Sharecrop$Ui$testId('submission-response')
						]),
					submission.responseJSON),
					$author$project$Main$validationErrorsView(submission.validationErrors)
				]));
	});
var $author$project$Main$submissionsList = function (state) {
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
			$author$project$Main$submissionRow(state),
			state.submissions));
};
var $author$project$Main$submissionsCard = function (state) {
	return $author$project$Sharecrop$Ui$card(
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Submissions'),
				$author$project$Main$reviewControls(state),
				$author$project$Main$submissionsList(state)
			]));
};
var $author$project$Main$SubmitClicked = {$: 'SubmitClicked'};
var $author$project$Main$SubmitInputChanged = function (a) {
	return {$: 'SubmitInputChanged', a: a};
};
var $author$project$Main$submitCard = function (state) {
	return A2(
		$elm$html$Html$form,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm'),
				$elm$html$Html$Events$onSubmit($author$project$Main$SubmitClicked)
			]),
		_List_fromArray(
			[
				$author$project$Sharecrop$Ui$sectionTitle('Submit a response'),
				$author$project$Sharecrop$Ui$textarea_(
				_List_fromArray(
					[
						$elm$html$Html$Attributes$placeholder('{}'),
						$elm$html$Html$Attributes$value(state.submitInput),
						$elm$html$Html$Events$onInput($author$project$Main$SubmitInputChanged),
						$elm$html$Html$Attributes$rows(6),
						$author$project$Sharecrop$Ui$testId('detail-submit-input')
					])),
				A2(
				$author$project$Sharecrop$Ui$primaryButton,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$type_('submit'),
						$author$project$Sharecrop$Ui$testId('detail-submit')
					]),
				'Submit response'),
				A2($author$project$Main$maybeNote, state.submitMessage, 'detail-submit-message')
			]));
};
var $author$project$Main$taskDetailPageView = F2(
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
		var backHref = isOwner ? '/tasks' : '/discovery';
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
						A2($author$project$Main$detailCard, origin, state)
					]),
				isOwner ? _List_fromArray(
					[
						$author$project$Main$ownerControlsCard(state),
						$author$project$Main$submissionsCard(state)
					]) : _List_fromArray(
					[
						$author$project$Main$reservationCard(state),
						$author$project$Main$submitCard(state)
					])));
	});
var $author$project$Main$TaskStateFilterChanged = function (a) {
	return {$: 'TaskStateFilterChanged', a: a};
};
var $author$project$Main$taskFilterButton = F2(
	function (selected, _v0) {
		var tag = _v0.a;
		var labelText = _v0.b;
		return A4(
			$author$project$Main$chooserButton,
			_Utils_eq(selected, tag),
			$author$project$Main$TaskStateFilterChanged(tag),
			'task-filter-' + ((tag === '') ? 'all' : tag),
			labelText);
	});
var $author$project$Main$taskStateFilterOptions = _List_fromArray(
	[
		_Utils_Tuple2('', 'All'),
		_Utils_Tuple2('open', 'Open'),
		_Utils_Tuple2('draft', 'Draft'),
		_Utils_Tuple2('closed', 'Closed')
	]);
var $author$project$Main$taskRow = function (item) {
	return A2(
		$elm$html$Html$div,
		_List_fromArray(
			[
				$elm$html$Html$Attributes$class('flex items-center justify-between py-2'),
				$author$project$Sharecrop$Ui$testId('task-row')
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
								$elm$html$Html$text(item.title)
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
								$author$project$Main$taskStateLabel(item.state) + (' · ' + (A3($author$project$Main$rewardLabel, item.rewardKind, item.rewardCreditAmount, item.rewardCollectibleCount) + $author$project$Main$activeAssigneeSuffix(item))))
							]))
					])),
				A2(
				$elm$html$Html$a,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$href('/tasks/' + item.id),
						$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
						$author$project$Sharecrop$Ui$testId('view-task')
					]),
				_List_fromArray(
					[
						$elm$html$Html$text('View')
					]))
			]));
};
var $author$project$Main$tasksList = function (tasks) {
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
		A2($elm$core$List$map, $author$project$Main$taskRow, tasks));
};
var $author$project$Main$tasksView = F2(
	function (origin, state) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					$author$project$Sharecrop$Ui$sectionTitle('My tasks'),
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
						$author$project$Main$taskFilterButton(state.taskStateFilter),
						$author$project$Main$taskStateFilterOptions)),
					$author$project$Main$tasksList(state.tasks)
				]));
	});
var $author$project$Main$teamDetailView = F2(
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
														$elm$html$Html$Attributes$href('/users/' + memberId),
														$elm$html$Html$Attributes$class('block py-2 text-sm underline'),
														$author$project$Sharecrop$Ui$testId('team-member-row')
													]),
												_List_fromArray(
													[
														$elm$html$Html$text(memberId)
													]));
										},
										detail.members))
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
				}()
				]));
	});
var $author$project$Main$userDetailView = F2(
	function (userId, state) {
		return $author$project$Sharecrop$Ui$card(
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
									$elm$html$Html$Attributes$href('/users/' + (userId + '/work')),
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
									$elm$html$Html$Attributes$href('/users/' + (userId + '/submissions')),
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
												$elm$html$Html$Attributes$href('/tasks/' + item.id),
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
				}()
				]));
	});
var $author$project$Main$userSubmissionsView = F2(
	function (userId, submissions) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('/users/' + userId),
							$elm$html$Html$Attributes$class($author$project$Sharecrop$Ui$secondaryButtonClass),
							$author$project$Sharecrop$Ui$testId('back-user')
						]),
					_List_fromArray(
						[
							$elm$html$Html$text('Back to profile')
						])),
					$author$project$Sharecrop$Ui$sectionTitle('Submissions'),
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
					A2(
						$elm$core$List$map,
						function (item) {
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
												$elm$html$Html$Attributes$href('/tasks/' + item.taskID),
												$elm$html$Html$Attributes$class('text-sm underline')
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
												$author$project$Main$submissionStateLabel(item.state))
											]))
									]));
						},
						submissions))
				]));
	});
var $author$project$Main$userTaskListView = F4(
	function (heading, identifier, userId, tasks) {
		return $author$project$Sharecrop$Ui$card(
			_List_fromArray(
				[
					A2(
					$elm$html$Html$a,
					_List_fromArray(
						[
							$elm$html$Html$Attributes$href('/users/' + userId),
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
										$elm$html$Html$Attributes$href('/tasks/' + item.id),
										$elm$html$Html$Attributes$class('block py-2 text-sm underline'),
										$author$project$Sharecrop$Ui$testId(identifier + '-row')
									]),
								_List_fromArray(
									[
										$elm$html$Html$text(
										item.title + (' · ' + $author$project$Main$taskStateLabel(item.state)))
									]));
						},
						tasks))
				]));
	});
var $author$project$Main$pageView = F2(
	function (origin, state) {
		var _v0 = state.page;
		switch (_v0.$) {
			case 'OverviewPage':
				return $author$project$Main$overviewView(state);
			case 'TasksPage':
				return A2($author$project$Main$tasksView, origin, state);
			case 'CreateTaskPage':
				return $author$project$Main$createTaskView(state);
			case 'TaskDetailPage':
				return A2($author$project$Main$taskDetailPageView, origin, state);
			case 'DiscoveryPage':
				return $author$project$Main$discoveryView(state);
			case 'FundingPage':
				return $author$project$Main$fundingView(state);
			case 'AgentsPage':
				return A2($author$project$Main$agentsView, origin, state);
			case 'CollectiblesPage':
				return $author$project$Main$collectiblesView(state);
			case 'OrganizationsPage':
				return $author$project$Main$organizationsView(state);
			case 'OrganizationDetailPage':
				return $author$project$Main$organizationDetailView(state);
			case 'UserDetailPage':
				var userId = _v0.a;
				return A2($author$project$Main$userDetailView, userId, state);
			case 'UserWorkPage':
				var userId = _v0.a;
				return A4($author$project$Main$userTaskListView, 'Public work', 'user-work', userId, state.userWork);
			case 'UserSubmissionsPage':
				var userId = _v0.a;
				return A2($author$project$Main$userSubmissionsView, userId, state.userSubmissions);
			case 'CollectibleDetailPage':
				var collectibleId = _v0.a;
				return A2($author$project$Main$collectibleDetailView, collectibleId, state);
			case 'SeriesDetailPage':
				var seriesId = _v0.a;
				return A2($author$project$Main$seriesDetailView, seriesId, state);
			default:
				var teamId = _v0.a;
				return A2($author$project$Main$teamDetailView, teamId, state);
		}
	});
var $author$project$Main$loggedInView = F2(
	function (origin, state) {
		return A2(
			$elm$html$Html$div,
			_List_fromArray(
				[
					$elm$html$Html$Attributes$class('space-y-6')
				]),
			_List_fromArray(
				[
					$author$project$Main$navBar(state.page),
					A2($author$project$Main$pageView, origin, state)
				]));
	});
var $author$project$Main$sessionView = function (model) {
	var _v0 = model.session;
	if (_v0.$ === 'LoggedOut') {
		return $author$project$Main$authView(model);
	} else {
		var state = _v0.a;
		return A2($author$project$Main$loggedInView, model.origin, state);
	}
};
var $author$project$Main$view = function (model) {
	return {
		body: _List_fromArray(
			[
				A2(
				$elm$html$Html$main_,
				_List_fromArray(
					[
						$elm$html$Html$Attributes$class('min-h-screen bg-slate-50 p-8 text-slate-950')
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
								$author$project$Main$sessionView(model)
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
					$author$project$Main$postRefresh);
			}),
		onUrlChange: $author$project$Main$UrlChanged,
		onUrlRequest: $author$project$Main$LinkClicked,
		subscriptions: function (_v0) {
			return $elm$core$Platform$Sub$none;
		},
		update: $author$project$Main$update,
		view: $author$project$Main$view
	});
_Platform_export({'Main':{'init':$author$project$Main$main(
	A2(
		$elm$json$Json$Decode$andThen,
		function (origin) {
			return $elm$json$Json$Decode$succeed(
				{origin: origin});
		},
		A2($elm$json$Json$Decode$field, 'origin', $elm$json$Json$Decode$string)))(0)}});}(this));